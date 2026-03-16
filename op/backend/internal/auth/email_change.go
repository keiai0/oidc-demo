package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/jwt"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// メールアドレス変更トークン有効期限（パスワードリセットより長め: 別デバイスでメールを確認する時間が必要）
const emailChangeTokenLifetime = 24 * time.Hour

// EmailChangeService はメールアドレス変更のリクエスト・実行を提供する。
type EmailChangeService struct {
	emailChangeTokenStore EmailChangeTokenStore
	userEmailUpdater      UserEmailUpdater
	emailSender           EmailSender
}

// NewEmailChangeService は EmailChangeService を生成する。
func NewEmailChangeService(
	emailChangeTokenStore EmailChangeTokenStore,
	userEmailUpdater UserEmailUpdater,
	emailSender EmailSender,
) *EmailChangeService {
	return &EmailChangeService{
		emailChangeTokenStore: emailChangeTokenStore,
		userEmailUpdater:      userEmailUpdater,
		emailSender:           emailSender,
	}
}

// RequestChange はメールアドレス変更を要求し、確認メールを送信する。
// ユーザーの現在のセッションが必要（認証済みユーザーのみ利用可能）。
func (s *EmailChangeService) RequestChange(ctx context.Context, userID uuid.UUID, newEmail string) error {
	// 既存の未使用トークンを無効化（前回のリクエストをキャンセル）
	if err := s.emailChangeTokenStore.InvalidateByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to invalidate existing tokens: %w", err)
	}

	// ランダムトークン生成（32バイト = 64文字 hex）
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := jwt.SHA256Hex(rawToken)

	changeToken := &model.EmailChangeToken{
		UserID:    userID,
		NewEmail:  newEmail,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(emailChangeTokenLifetime),
	}
	if err := s.emailChangeTokenStore.Create(ctx, changeToken); err != nil {
		return fmt.Errorf("failed to create email change token: %w", err)
	}

	// メール送信（スタブ: コンソール出力）
	if err := s.emailSender.SendEmailChangeEmail(ctx, newEmail, rawToken); err != nil {
		return fmt.Errorf("failed to send email change verification: %w", err)
	}

	return nil
}

// VerifyChange はメールアドレス変更トークンを検証し、メールアドレスを更新する。
func (s *EmailChangeService) VerifyChange(ctx context.Context, rawToken string) error {
	tokenHash := jwt.SHA256Hex(rawToken)

	changeToken, err := s.emailChangeTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to find email change token: %w", err)
	}
	if changeToken == nil || changeToken.IsUsed() || changeToken.IsExpired() {
		return ErrEmailChangeTokenInvalid
	}

	// メールアドレスを更新し email_verified を true にする
	if err := s.userEmailUpdater.UpdateEmail(ctx, changeToken.UserID, changeToken.NewEmail); err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	// トークンを使用済みにする
	if err := s.emailChangeTokenStore.MarkAsUsed(ctx, changeToken.ID); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	return nil
}
