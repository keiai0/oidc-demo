package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type AuthService struct {
	tenantFinder    TenantFinder
	userFinder      UserFinder
	sessionStore    SessionStore
	loginTracker    LoginAttemptTracker
	verifyPassword  PasswordVerifyFunc
	mfaConfigFinder MfaConfigFinder
}

func NewAuthService(
	tenantFinder TenantFinder,
	userFinder UserFinder,
	sessionStore SessionStore,
	loginTracker LoginAttemptTracker,
	verifyPassword PasswordVerifyFunc,
	mfaConfigFinder MfaConfigFinder,
) *AuthService {
	return &AuthService{
		tenantFinder:    tenantFinder,
		userFinder:      userFinder,
		sessionStore:    sessionStore,
		loginTracker:    loginTracker,
		verifyPassword:  verifyPassword,
		mfaConfigFinder: mfaConfigFinder,
	}
}

func (s *AuthService) Login(ctx context.Context, input *model.LoginInput) (*model.LoginOutput, error) {
	// テナント検索
	tenant, err := s.tenantFinder.FindByCode(ctx, input.TenantCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidCredentials
	}

	// ユーザー検索 (Credentials preload 済み)
	user, err := s.userFinder.FindByTenantAndLoginID(ctx, tenant.ID, input.LoginID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != "active" {
		return nil, ErrInvalidCredentials
	}

	// アカウントロックチェック
	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// パスワード検証
	passwordHash := findPasswordHash(user.Credentials)
	if passwordHash == "" {
		return nil, ErrInvalidCredentials
	}

	match, err := s.verifyPassword(input.Password, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !match {
		// 失敗回数をインクリメント（エラーは無視しない）
		if trackErr := s.loginTracker.IncrementFailedLogin(ctx, user.ID); trackErr != nil {
			return nil, fmt.Errorf("failed to track login attempt: %w", trackErr)
		}
		return nil, ErrInvalidCredentials
	}

	// ログイン成功時に失敗カウンターをリセット
	if user.FailedLoginCount > 0 {
		if resetErr := s.loginTracker.ResetFailedLogin(ctx, user.ID); resetErr != nil {
			return nil, fmt.Errorf("failed to reset login attempts: %w", resetErr)
		}
	}

	// MFA 判定: ユーザーが MFA 設定済み、またはテナントが MFA 強制
	mfaRequired := false
	mfaSetupRequired := false
	passkeyRegistered := false
	var mfaMethods []string
	if s.mfaConfigFinder != nil {
		mfaConfigs, mfaErr := s.mfaConfigFinder.FindAllEnabledByUserID(ctx, user.ID)
		if mfaErr != nil {
			return nil, fmt.Errorf("failed to check MFA config: %w", mfaErr)
		}
		for _, mc := range mfaConfigs {
			if mc.Type == "webauthn" {
				// パスキーは MFA ではなくパスワードレス認証なので MFA 判定に含めない
				passkeyRegistered = true
			} else {
				// TOTP 等は MFA として扱う
				mfaRequired = true
				mfaMethods = append(mfaMethods, mc.Type)
			}
		}
		if !mfaRequired && tenant.MfaRequired {
			// テナントが MFA 強制 + ユーザー未設定 → セットアップが必要
			mfaSetupRequired = true
		}
	}

	pendingMFA := mfaRequired || mfaSetupRequired

	// セッション作成
	now := time.Now()
	session := &model.Session{
		UserID:           user.ID,
		TenantID:         tenant.ID,
		IPAddress:        input.IPAddress,
		UserAgent:        input.UserAgent,
		AuthTime:         now,
		AMR:              model.StringSlice{"pwd"},
		ACR:              "urn:mace:incommon:iap:bronze",
		PendingMFA:       pendingMFA,
		MfaSetupRequired: mfaSetupRequired,
		ExpiresAt:        now.Add(time.Duration(tenant.SessionLifetime) * time.Second),
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// last_login_at 更新
	_ = s.userFinder.UpdateLastLoginAt(ctx, user.ID, now)

	return &model.LoginOutput{
		SessionID:        session.ID,
		User:             user,
		MFARequired:      mfaRequired,
		MFASetupRequired: mfaSetupRequired,
		MFAMethods:        mfaMethods,
		PasskeyRegistered: passkeyRegistered,
	}, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, sessionID uuid.UUID) (*model.Session, error) {
	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if !session.IsValid() {
		return nil, ErrSessionExpired
	}
	return session, nil
}

func findPasswordHash(credentials []model.Credential) string {
	for _, cred := range credentials {
		if cred.Type == "password" && cred.PasswordCredential != nil {
			return cred.PasswordCredential.PasswordHash
		}
	}
	return ""
}
