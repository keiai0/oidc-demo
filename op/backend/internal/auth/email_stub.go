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
