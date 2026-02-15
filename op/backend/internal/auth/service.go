package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type AuthService struct {
	tenantFinder   TenantFinder
	userFinder     UserFinder
	sessionStore   SessionStore
	verifyPassword PasswordVerifyFunc
}

func NewAuthService(
	tenantFinder TenantFinder,
	userFinder UserFinder,
	sessionStore SessionStore,
	verifyPassword PasswordVerifyFunc,
) *AuthService {
	return &AuthService{
		tenantFinder:   tenantFinder,
		userFinder:     userFinder,
		sessionStore:   sessionStore,
		verifyPassword: verifyPassword,
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
		return nil, ErrInvalidCredentials
	}

	// セッション作成
	session := &model.Session{
		UserID:    user.ID,
		TenantID:  tenant.ID,
		IPAddress: input.IPAddress,
		UserAgent: input.UserAgent,
		ExpiresAt: time.Now().Add(time.Duration(tenant.SessionLifetime) * time.Second),
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// last_login_at 更新
	now := time.Now()
	_ = s.userFinder.UpdateLastLoginAt(ctx, user.ID, now)

	return &model.LoginOutput{
		SessionID: session.ID,
		User:      user,
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
