package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// PasskeyLoginService はパスキーによるパスワードレスログインのビジネスロジックを提供する。
type PasskeyLoginService struct {
	webauthn        *gowebauthn.WebAuthn
	mfaConfigFinder MfaConfigFinder
	credStore       WebAuthnCredentialStore
	sessionStore    SessionStore
	userFinder      UserFinder
	tenantFinder    TenantFinder

	// チャレンジの一時保存（未ログイン状態ではセッションに保存できないため）
	challenges sync.Map // map[string]*gowebauthn.SessionData
}

// NewPasskeyLoginService は PasskeyLoginService を生成する。
func NewPasskeyLoginService(
	wa *gowebauthn.WebAuthn,
	mfaConfigFinder MfaConfigFinder,
	credStore WebAuthnCredentialStore,
	sessionStore SessionStore,
	userFinder UserFinder,
	tenantFinder TenantFinder,
) *PasskeyLoginService {
	return &PasskeyLoginService{
		webauthn:        wa,
		mfaConfigFinder: mfaConfigFinder,
		credStore:       credStore,
		sessionStore:    sessionStore,
		userFinder:      userFinder,
		tenantFinder:    tenantFinder,
	}
}

// BeginPasskeyLogin は Discoverable Login のチャレンジを発行する。
func (s *PasskeyLoginService) BeginPasskeyLogin(ctx context.Context) (optionsJSON json.RawMessage, challengeID string, err error) {
	assertion, sessionData, err := s.webauthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin discoverable login: %w", err)
	}

	// チャレンジ ID を生成してインメモリに保存
	challengeID = uuid.New().String()
	s.challenges.Store(challengeID, sessionData)

	// 60秒後に自動削除
	go func() {
		time.Sleep(60 * time.Second)
		s.challenges.Delete(challengeID)
	}()

	optionsJSON, err = json.Marshal(assertion)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal assertion options: %w", err)
	}

	return optionsJSON, challengeID, nil
}

// CompletePasskeyLogin はパスキー認証の Assertion を検証し、セッションを作成する。
func (s *PasskeyLoginService) CompletePasskeyLogin(
	ctx context.Context,
	challengeID string,
	tenantCode string,
	r *http.Request,
	ipAddress string,
	userAgent string,
) (*model.LoginOutput, error) {
	// チャレンジを復元
	sessionDataRaw, ok := s.challenges.LoadAndDelete(challengeID)
	if !ok {
		return nil, fmt.Errorf("challenge not found or expired")
	}
	sessionData := sessionDataRaw.(*gowebauthn.SessionData)

	// DiscoverableUserHandler: userHandle からユーザーを特定し WebAuthn User を返す
	handler := func(rawID, userHandle []byte) (gowebauthn.User, error) {
		// userHandle は登録時に WebAuthnID() で返した値 = user.ID (uuid 16 bytes)
		userID, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, fmt.Errorf("invalid user handle: %w", err)
		}

		mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "webauthn")
		if err != nil {
			return nil, fmt.Errorf("failed to find WebAuthn config: %w", err)
		}
		if mfaConfig == nil || !mfaConfig.Enabled || len(mfaConfig.WebAuthnCredentials) == 0 {
			return nil, fmt.Errorf("no WebAuthn credentials for user")
		}

		return &webauthnUser{
			id:          userID,
			credentials: toWebAuthnCredentials(mfaConfig.WebAuthnCredentials),
		}, nil
	}

	// Assertion 検証 (FinishPasskeyLogin は User も返す)
	webUser, credential, err := s.webauthn.FinishPasskeyLogin(handler, *sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("failed to finish passkey login: %w", err)
	}

	// クローン検知 (WebAuthn Level 2 Section 7.2 step 17)
	if credential.Authenticator.CloneWarning {
		return nil, ErrWebAuthnCloneDetected
	}

	// ユーザー ID を取得
	userID, err := uuid.FromBytes(webUser.WebAuthnID())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID from WebAuthn: %w", err)
	}

	// sign_count 更新
	credIDStr := base64.RawURLEncoding.EncodeToString(credential.ID)
	dbCred, err := s.credStore.FindByCredentialID(ctx, credIDStr)
	if err != nil || dbCred == nil {
		return nil, fmt.Errorf("credential not found in database")
	}
	if err := s.credStore.UpdateSignCount(ctx, dbCred.ID, credential.Authenticator.SignCount); err != nil {
		return nil, fmt.Errorf("failed to update sign count: %w", err)
	}

	// ユーザー情報取得
	user, err := s.userFinder.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.Status != "active" {
		return nil, ErrInvalidCredentials
	}
	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// テナント検索
	tenant, err := s.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// セッション作成（パスキー認証完了済み、MFA 不要）
	now := time.Now()
	session := &model.Session{
		UserID:    user.ID,
		TenantID:  tenant.ID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		AuthTime:  now,
		AMR:       model.StringSlice{"hwk"},
		ACR:       "urn:mace:incommon:iap:silver",
		PendingMFA:       false,
		MfaSetupRequired: false,
		ExpiresAt:        now.Add(time.Duration(tenant.SessionLifetime) * time.Second),
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// last_login_at 更新
	_ = s.userFinder.UpdateLastLoginAt(ctx, user.ID, now)

	return &model.LoginOutput{
		SessionID:         session.ID,
		User:              user,
		PasskeyRegistered: true,
	}, nil
}
