package auth

import (
	"context"
	"log"
)

// StubEmailSender はメール送信をコンソールログに出力するスタブ実装。
// 開発・検証用途。本番ではメール配信サービス（SendGrid, SES 等）に差し替える。
type StubEmailSender struct{}

// NewStubEmailSender は StubEmailSender を生成する。
func NewStubEmailSender() *StubEmailSender {
	return &StubEmailSender{}
}

// SendPasswordResetEmail はリセットトークンをログに出力する。
func (s *StubEmailSender) SendPasswordResetEmail(_ context.Context, email, token string) error {
	log.Printf("[STUB EMAIL] Password reset token for %s: %s", email, token)
	return nil
}

// SendEmailChangeEmail はメールアドレス変更確認トークンをログに出力する。
func (s *StubEmailSender) SendEmailChangeEmail(_ context.Context, newEmail, token string) error {
	log.Printf("[STUB EMAIL] Email change verification token for %s: %s", newEmail, token)
	return nil
}
