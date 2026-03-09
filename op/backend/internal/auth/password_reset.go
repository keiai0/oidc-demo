package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/jwt"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// リセットトークン有効期限 (OWASP 推奨: 30分以内)
const resetTokenLifetime = 30 * time.Minute

var (
	ErrResetTokenInvalid = errors.New("reset token is invalid or expired")
)

// PasswordResetService はパスワードリセットのリクエスト・実行を提供する。
type PasswordResetService struct {
	tenantFinder   TenantFinder
	userFinder     UserFinderByEmail
	resetTokenStore PasswordResetTokenStore
	passwordSvc    *PasswordService
	loginTracker   LoginAttemptTracker
	emailSender    EmailSender
}

// NewPasswordResetService は PasswordResetService を生成する。
func NewPasswordResetService(
	tenantFinder TenantFinder,
	userFinder UserFinderByEmail,
	resetTokenStore PasswordResetTokenStore,
	passwordSvc *PasswordService,
	loginTracker LoginAttemptTracker,
	emailSender EmailSender,
) *PasswordResetService {
	return &PasswordResetService{
		tenantFinder:    tenantFinder,
		userFinder:      userFinder,
		resetTokenStore: resetTokenStore,
		passwordSvc:     passwordSvc,
		loginTracker:    loginTracker,
		emailSender:     emailSender,
	}
}

// RequestReset はパスワードリセットを要求する。
// ユーザーが存在しない場合でもエラーを返さない（OWASP: ユーザー列挙防止）。
func (s *PasswordResetService) RequestReset(ctx context.Context, tenantCode, email string) error {
	tenant, err := s.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		return fmt.Errorf("failed to find tenant: %w", err)
	}
	if tenant == nil {
		return nil // テナント不存在でもエラーを返さない
	}

	user, err := s.userFinder.FindByTenantAndEmail(ctx, tenant.ID, email)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil // ユーザー不存在でもエラーを返さない
	}

	// 既存の未使用トークンを無効化
	if err := s.resetTokenStore.InvalidateByUserID(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to invalidate existing tokens: %w", err)
	}

	// ランダムトークン生成（32バイト = 64文字 hex）
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := jwt.SHA256Hex(rawToken)

	resetToken := &model.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(resetTokenLifetime),
	}
	if err := s.resetTokenStore.Create(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// メール送信（スタブ: コンソール出力）
	if err := s.emailSender.SendPasswordResetEmail(ctx, email, rawToken); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	return nil
}

// ExecuteReset はリセットトークンを検証し、パスワードを変更する。
func (s *PasswordResetService) ExecuteReset(ctx context.Context, rawToken, newPassword string) error {
	tokenHash := jwt.SHA256Hex(rawToken)

	resetToken, err := s.resetTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to find reset token: %w", err)
	}
	if resetToken == nil {
		return ErrResetTokenInvalid
	}
	if resetToken.IsUsed() || resetToken.IsExpired() {
		return ErrResetTokenInvalid
	}

	// パスワード履歴チェック + 更新（PasswordService のロジックを再利用）
	newHash, err := s.passwordSvc.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// パスワード履歴チェック
	histories, err := s.passwordSvc.historyStore.FindRecentByUserID(ctx, resetToken.UserID, passwordHistoryLimit)
	if err != nil {
		return fmt.Errorf("failed to find password history: %w", err)
	}
	for _, h := range histories {
		match, verifyErr := s.passwordSvc.verifyPassword(newPassword, h.PasswordHash)
		if verifyErr != nil {
			continue
		}
		if match {
			return ErrPasswordInHistory
		}
	}

	// ユーザーのクレデンシャルを取得してパスワード更新
	user, err := s.passwordSvc.userFinder.FindByIDWithCredentials(ctx, resetToken.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	cred, currentHash := findPasswordCredential(user.Credentials)
	if cred == nil {
		return ErrInvalidCredentials
	}

	// 現在のパスワードと同じでないか確認
	sameAsCurrent, err := s.passwordSvc.verifyPassword(newPassword, currentHash)
	if err != nil {
		return fmt.Errorf("failed to verify against current password: %w", err)
	}
	if sameAsCurrent {
		return ErrPasswordSameAsCurrent
	}

	// パスワード更新
	if err := s.passwordSvc.passwordUpdater.UpdateHash(ctx, cred.ID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// パスワード履歴に記録
	history := &model.PasswordHistory{
		UserID:       resetToken.UserID,
		PasswordHash: newHash,
	}
	if err := s.passwordSvc.historyStore.Create(ctx, history); err != nil {
		return fmt.Errorf("failed to create password history: %w", err)
	}

	// トークンを使用済みにする
	if err := s.resetTokenStore.MarkAsUsed(ctx, resetToken.ID); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// 失敗カウンターをリセット
	_ = s.loginTracker.ResetFailedLogin(ctx, resetToken.UserID)

	return nil
}
