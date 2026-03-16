package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// バックアップコードの生成数
const backupCodeCount = 10

// BackupCodeService は MFA バックアップコードの生成・検証を提供する。
type BackupCodeService struct {
	backupCodeStore BackupCodeStore
	sessionUpdater  SessionUpdater
	hashPassword    HashPasswordFunc
	verifyPassword  PasswordVerifyFunc
}

// NewBackupCodeService は BackupCodeService を生成する。
func NewBackupCodeService(
	backupCodeStore BackupCodeStore,
	sessionUpdater SessionUpdater,
	hashPassword HashPasswordFunc,
	verifyPassword PasswordVerifyFunc,
) *BackupCodeService {
	return &BackupCodeService{
		backupCodeStore: backupCodeStore,
		sessionUpdater:  sessionUpdater,
		hashPassword:    hashPassword,
		verifyPassword:  verifyPassword,
	}
}

// Generate はバックアップコードを10個生成し、argon2id ハッシュをDBに保存して平文を返す。
// 既存のバックアップコードは全て削除される。平文コードはこの1回限り返却される。
func (s *BackupCodeService) Generate(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// 旧コードを全削除
	if err := s.backupCodeStore.DeleteByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to delete existing backup codes: %w", err)
	}

	plainCodes := make([]string, 0, backupCodeCount)
	for i := 0; i < backupCodeCount; i++ {
		// 6バイトランダム → 12文字 hex → "xxxxxx-xxxxxx" 形式
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		encoded := hex.EncodeToString(b)
		plain := encoded[:6] + "-" + encoded[6:]

		hash, err := s.hashPassword(plain)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}

		code := &model.BackupCode{
			UserID:   userID,
			CodeHash: hash,
		}
		if err := s.backupCodeStore.Create(ctx, code); err != nil {
			return nil, fmt.Errorf("failed to save backup code: %w", err)
		}

		plainCodes = append(plainCodes, plain)
	}

	return plainCodes, nil
}

// Verify はバックアップコードを検証し、一致したコードを使用済みにする。
// session.PendingMFA が true の場合、MFA 完了としてセッションを更新する。
func (s *BackupCodeService) Verify(ctx context.Context, session *model.Session, rawCode string) error {
	codes, err := s.backupCodeStore.FindUnusedByUserID(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("failed to find backup codes: %w", err)
	}

	for _, bc := range codes {
		match, err := s.verifyPassword(rawCode, bc.CodeHash)
		if err != nil {
			// argon2id 検証エラーは無視して次のコードを試す
			continue
		}
		if !match {
			continue
		}

		// 一致したコードを使用済みにする
		if err := s.backupCodeStore.MarkAsUsed(ctx, bc.ID); err != nil {
			return fmt.Errorf("failed to mark backup code as used: %w", err)
		}

		// MFA 待ちセッションを完了状態に更新する
		// AMR: バックアップコードは特定の MFA 方式ではないため汎用的な "mfa" を使用
		if session.PendingMFA {
			amr := model.StringSlice{"pwd", "mfa"}
			acr := "urn:mace:incommon:iap:silver"
			if err := s.sessionUpdater.UpdateMFACompleted(ctx, session.ID, amr, acr); err != nil {
				return fmt.Errorf("failed to update session MFA state: %w", err)
			}
		}

		return nil
	}

	return ErrBackupCodeInvalid
}
