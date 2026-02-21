package management

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

const adminSessionLifetime = 8 * time.Hour

var (
	// ErrAdminInvalidCredentials はログイン ID またはパスワードが不正、もしくはユーザーが無効であることを示す。
	ErrAdminInvalidCredentials = errors.New("invalid admin credentials")
	// ErrAdminSessionNotFound はセッションが存在しないことを示す。
	ErrAdminSessionNotFound = errors.New("admin session not found")
	// ErrAdminSessionExpired はセッションが期限切れまたは失効済みであることを示す。
	ErrAdminSessionExpired = errors.New("admin session expired or revoked")
)

// AdminAuthService は管理者の認証とセッション管理を行う。
type AdminAuthService struct {
	userFinder     AdminUserFinder
	sessionStore   AdminSessionStore
	verifyPassword PasswordVerifyFunc
}

// NewAdminAuthService は AdminAuthService を生成する。
func NewAdminAuthService(
	userFinder AdminUserFinder,
	sessionStore AdminSessionStore,
	verifyPassword PasswordVerifyFunc,
) *AdminAuthService {
	return &AdminAuthService{
		userFinder:     userFinder,
		sessionStore:   sessionStore,
		verifyPassword: verifyPassword,
	}
}

// Login は管理者ユーザーを認証し、セッションを作成する。
func (s *AdminAuthService) Login(ctx context.Context, loginID, password, ipAddress, userAgent string) (*model.AdminSession, *model.AdminUser, error) {
	user, err := s.userFinder.FindByLoginID(ctx, loginID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find admin user: %w", err)
	}
	if user == nil {
		return nil, nil, ErrAdminInvalidCredentials
	}

	if user.Status != "active" {
		return nil, nil, ErrAdminInvalidCredentials
	}

	match, err := s.verifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !match {
		return nil, nil, ErrAdminInvalidCredentials
	}

	session := &model.AdminSession{
		AdminUserID: user.ID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		ExpiresAt:   time.Now().Add(adminSessionLifetime),
	}

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to create admin session: %w", err)
	}

	now := time.Now()
	_ = s.userFinder.UpdateLastLoginAt(ctx, user.ID, now)

	return session, user, nil
}

// RevokeSession は指定 ID の管理者セッションを失効させる。
func (s *AdminAuthService) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.sessionStore.Revoke(ctx, sessionID)
}

// ValidateSession は指定 ID の管理者セッションを検証する。
func (s *AdminAuthService) ValidateSession(ctx context.Context, sessionID uuid.UUID) (*model.AdminSession, error) {
	session, err := s.sessionStore.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find admin session: %w", err)
	}
	if session == nil {
		return nil, ErrAdminSessionNotFound
	}
	if !session.IsValid() {
		return nil, ErrAdminSessionExpired
	}
	return session, nil
}
