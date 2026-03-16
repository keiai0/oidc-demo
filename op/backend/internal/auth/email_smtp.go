package auth

import (
	"context"
	"fmt"
	"net/smtp"
)

// SMTPEmailSender は SMTP を使用してメールを送信する実装。
// 開発環境では MailHog（ポート 1025）と組み合わせて使用する。
type SMTPEmailSender struct {
	host            string
	port            string
	from            string
	frontendBaseURL string
}

// NewSMTPEmailSender は SMTPEmailSender を生成する。
func NewSMTPEmailSender(host, port, from, frontendBaseURL string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:            host,
		port:            port,
		from:            from,
		frontendBaseURL: frontendBaseURL,
	}
}

// SendPasswordResetEmail はパスワードリセット用のメールを送信する。
func (s *SMTPEmailSender) SendPasswordResetEmail(_ context.Context, email, token string) error {
	subject := "パスワードリセットのご案内"
	resetURL := fmt.Sprintf("%s/password-reset?token=%s", s.frontendBaseURL, token)
	body := fmt.Sprintf("以下のリンクからパスワードをリセットしてください。\n\n%s\n\nこのリンクは24時間有効です。\n身に覚えのない場合は本メールを無視してください。", resetURL)
	return s.send(email, subject, body)
}

// SendEmailChangeEmail はメールアドレス変更確認メールを送信する。
func (s *SMTPEmailSender) SendEmailChangeEmail(_ context.Context, newEmail, token string) error {
	subject := "メールアドレス変更の確認"
	verifyURL := fmt.Sprintf("%s/email-verify?token=%s", s.frontendBaseURL, token)
	body := fmt.Sprintf("以下のリンクからメールアドレスの変更を確認してください。\n\n%s\n\nこのリンクは24時間有効です。\n身に覚えのない場合は本メールを無視してください。", verifyURL)
	return s.send(newEmail, subject, body)
}

// send は指定した宛先にメールを送信する。
func (s *SMTPEmailSender) send(to, subject, body string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body,
	)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	// MailHog は認証不要のため auth=nil で送信
	return smtp.SendMail(addr, nil, s.from, []string{to}, []byte(msg))
}
