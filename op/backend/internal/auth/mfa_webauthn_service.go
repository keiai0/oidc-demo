package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

var (
	ErrWebAuthnCredentialNotFound = errors.New("WebAuthn credential not found")
	ErrWebAuthnCloneDetected      = errors.New("WebAuthn authenticator clone detected")
	ErrWebAuthnNoCredentials      = errors.New("no WebAuthn credentials registered")
)

// MFAWebAuthnService は WebAuthn MFA のビジネスロジックを提供する。
type MFAWebAuthnService struct {
	webauthn         *gowebauthn.WebAuthn
	mfaConfigFinder  MfaConfigFinder
	mfaConfigStore   MfaConfigStore
	credStore        WebAuthnCredentialStore
	sessionUpdater   SessionUpdater
	challengeUpdater WebAuthnChallengeUpdater
}

// NewMFAWebAuthnService は MFAWebAuthnService を生成する。
func NewMFAWebAuthnService(
	wa *gowebauthn.WebAuthn,
	mfaConfigFinder MfaConfigFinder,
	mfaConfigStore MfaConfigStore,
	credStore WebAuthnCredentialStore,
	sessionUpdater SessionUpdater,
	challengeUpdater WebAuthnChallengeUpdater,
) *MFAWebAuthnService {
	return &MFAWebAuthnService{
		webauthn:         wa,
		mfaConfigFinder:  mfaConfigFinder,
		mfaConfigStore:   mfaConfigStore,
		credStore:        credStore,
		sessionUpdater:   sessionUpdater,
		challengeUpdater: challengeUpdater,
	}
}

// BeginRegistration は WebAuthn 登録のチャレンジを発行する。
func (s *MFAWebAuthnService) BeginRegistration(ctx context.Context, userID uuid.UUID, userName, displayName string, session *model.Session) (json.RawMessage, error) {
	// 既存の webauthn MFA 設定を確認
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "webauthn")
	if err != nil {
		return nil, fmt.Errorf("failed to check existing MFA config: %w", err)
	}

	// MFA 設定がなければ作成
	if mfaConfig == nil {
		mfaConfig = &model.MfaConfig{
			UserID:  userID,
			Type:    "webauthn",
			Enabled: false,
		}
		if err := s.mfaConfigStore.Create(ctx, mfaConfig); err != nil {
			return nil, fmt.Errorf("failed to create MFA config: %w", err)
		}
	}

	// 既存クレデンシャルを取得（除外リスト用）
	existingCreds := toWebAuthnCredentials(mfaConfig.WebAuthnCredentials)

	user := &webauthnUser{
		id:          userID,
		name:        userName,
		displayName: displayName,
		credentials: existingCreds,
	}

	creation, sessionData, err := s.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	// セッションデータを JSON にして保存
	challengeJSON, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session data: %w", err)
	}
	challengeStr := string(challengeJSON)
	if err := s.challengeUpdater.UpdateWebAuthnChallenge(ctx, session.ID, &challengeStr); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	// CredentialCreation を JSON にして返す
	creationJSON, err := json.Marshal(creation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal creation options: %w", err)
	}

	return creationJSON, nil
}

// CompleteRegistration は WebAuthn 登録の Attestation を検証し、クレデンシャルを保存する。
func (s *MFAWebAuthnService) CompleteRegistration(ctx context.Context, userID uuid.UUID, session *model.Session, r *http.Request, credName string) error {
	// チャレンジを復元
	if session.WebAuthnChallenge == nil {
		return fmt.Errorf("no WebAuthn challenge found in session")
	}
	var sessionData gowebauthn.SessionData
	if err := json.Unmarshal([]byte(*session.WebAuthnChallenge), &sessionData); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// MFA 設定を取得
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "webauthn")
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil {
		return ErrMFANotConfigured
	}

	existingCreds := toWebAuthnCredentials(mfaConfig.WebAuthnCredentials)
	user := &webauthnUser{
		id:          userID,
		name:        "",
		displayName: "",
		credentials: existingCreds,
	}

	// Attestation 検証
	credential, err := s.webauthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		return fmt.Errorf("failed to finish WebAuthn registration: %w", err)
	}

	// DB に保存
	webauthnCred := &model.WebAuthnCredential{
		MfaConfigID:     mfaConfig.ID,
		CredentialID:    base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:       base64.StdEncoding.EncodeToString(credential.PublicKey),
		AttestationType: credential.AttestationType,
		AAGUID:          base64.StdEncoding.EncodeToString(credential.Authenticator.AAGUID),
		SignCount:       credential.Authenticator.SignCount,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		Name:            credName,
	}
	if err := s.credStore.Create(ctx, webauthnCred); err != nil {
		return fmt.Errorf("failed to save WebAuthn credential: %w", err)
	}

	// MFA が未有効なら有効化
	if !mfaConfig.Enabled {
		now := time.Now()
		mfaConfig.Enabled = true
		mfaConfig.VerifiedAt = &now
		if err := s.mfaConfigStore.Update(ctx, mfaConfig); err != nil {
			return fmt.Errorf("failed to enable MFA config: %w", err)
		}
	}

	// チャレンジをクリア
	if err := s.challengeUpdater.UpdateWebAuthnChallenge(ctx, session.ID, nil); err != nil {
		return fmt.Errorf("failed to clear challenge: %w", err)
	}

	return nil
}

// BeginAuthentication は WebAuthn 認証のチャレンジを発行する。
func (s *MFAWebAuthnService) BeginAuthentication(ctx context.Context, session *model.Session) (json.RawMessage, error) {
	// ユーザーの webauthn 設定を取得
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, session.UserID, "webauthn")
	if err != nil {
		return nil, fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil || !mfaConfig.Enabled || len(mfaConfig.WebAuthnCredentials) == 0 {
		return nil, ErrWebAuthnNoCredentials
	}

	existingCreds := toWebAuthnCredentials(mfaConfig.WebAuthnCredentials)
	user := &webauthnUser{
		id:          session.UserID,
		name:        "",
		displayName: "",
		credentials: existingCreds,
	}

	assertion, sessionData, err := s.webauthn.BeginLogin(user)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn login: %w", err)
	}

	// セッションデータを保存
	challengeJSON, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session data: %w", err)
	}
	challengeStr := string(challengeJSON)
	if err := s.challengeUpdater.UpdateWebAuthnChallenge(ctx, session.ID, &challengeStr); err != nil {
		return nil, fmt.Errorf("failed to save challenge: %w", err)
	}

	assertionJSON, err := json.Marshal(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal assertion options: %w", err)
	}

	return assertionJSON, nil
}

// CompleteAuthentication は WebAuthn 認証の Assertion を検証し、セッションを更新する。
func (s *MFAWebAuthnService) CompleteAuthentication(ctx context.Context, session *model.Session, r *http.Request) error {
	if !session.PendingMFA {
		return ErrMFANotPending
	}

	// チャレンジを復元
	if session.WebAuthnChallenge == nil {
		return fmt.Errorf("no WebAuthn challenge found in session")
	}
	var sessionData gowebauthn.SessionData
	if err := json.Unmarshal([]byte(*session.WebAuthnChallenge), &sessionData); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// ユーザーのクレデンシャルを取得
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, session.UserID, "webauthn")
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil || !mfaConfig.Enabled {
		return ErrMFANotConfigured
	}

	existingCreds := toWebAuthnCredentials(mfaConfig.WebAuthnCredentials)
	user := &webauthnUser{
		id:          session.UserID,
		name:        "",
		displayName: "",
		credentials: existingCreds,
	}

	// Assertion 検証
	credential, err := s.webauthn.FinishLogin(user, sessionData, r)
	if err != nil {
		return fmt.Errorf("failed to finish WebAuthn login: %w", err)
	}

	// クローン検知: sign_count が後退していたら拒否
	// WebAuthn Level 2 Section 7.2 step 17
	if credential.Authenticator.CloneWarning {
		return ErrWebAuthnCloneDetected
	}

	// DB のクレデンシャルを特定し sign_count を更新
	credIDStr := base64.RawURLEncoding.EncodeToString(credential.ID)
	for _, mc := range mfaConfig.WebAuthnCredentials {
		if mc.CredentialID == credIDStr {
			if err := s.credStore.UpdateSignCount(ctx, mc.ID, credential.Authenticator.SignCount); err != nil {
				return fmt.Errorf("failed to update sign count: %w", err)
			}
			break
		}
	}

	// チャレンジをクリア
	if err := s.challengeUpdater.UpdateWebAuthnChallenge(ctx, session.ID, nil); err != nil {
		return fmt.Errorf("failed to clear challenge: %w", err)
	}

	// セッション更新: pending_mfa=false, AMR=["pwd","hwk"], ACR=silver
	amr := model.StringSlice{"pwd", "hwk"}
	acr := "urn:mace:incommon:iap:silver"
	if err := s.sessionUpdater.UpdateMFACompleted(ctx, session.ID, amr, acr); err != nil {
		return fmt.Errorf("failed to update session MFA status: %w", err)
	}

	return nil
}

// ListCredentials はユーザーの WebAuthn クレデンシャル一覧を返す。
func (s *MFAWebAuthnService) ListCredentials(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error) {
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "webauthn")
	if err != nil {
		return nil, fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil {
		return []model.WebAuthnCredential{}, nil
	}
	return mfaConfig.WebAuthnCredentials, nil
}

// DeleteCredential はユーザーの WebAuthn クレデンシャルを削除する。
// 残りのクレデンシャルがなくなった場合、MFA 設定も削除する。
func (s *MFAWebAuthnService) DeleteCredential(ctx context.Context, userID uuid.UUID, credentialDBID uuid.UUID) error {
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "webauthn")
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil {
		return ErrMFANotConfigured
	}

	// 対象のクレデンシャルがこのユーザーのものか確認
	found := false
	for _, c := range mfaConfig.WebAuthnCredentials {
		if c.ID == credentialDBID {
			found = true
			break
		}
	}
	if !found {
		return ErrWebAuthnCredentialNotFound
	}

	if err := s.credStore.Delete(ctx, credentialDBID); err != nil {
		return fmt.Errorf("failed to delete WebAuthn credential: %w", err)
	}

	// 残りのクレデンシャル数を確認
	remaining, err := s.credStore.FindByMfaConfigID(ctx, mfaConfig.ID)
	if err != nil {
		return fmt.Errorf("failed to count remaining credentials: %w", err)
	}
	if len(remaining) == 0 {
		// クレデンシャルがなくなったので MFA 設定も削除
		if err := s.mfaConfigStore.Delete(ctx, mfaConfig.ID); err != nil {
			return fmt.Errorf("failed to delete MFA config: %w", err)
		}
	}

	return nil
}
