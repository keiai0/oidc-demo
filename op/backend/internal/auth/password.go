package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// パスワード履歴チェック件数 (OWASP 推奨)
const passwordHistoryLimit = 5

var (
	ErrPasswordInHistory = errors.New("password was recently used")
	ErrPasswordSameAsCurrent = errors.New("new password must differ from current password")
)

// PasswordService はパスワード変更・履歴チェックを提供する。
type PasswordService struct {
	userFinder      UserFinderWithCredentials
	historyStore    PasswordHistoryStore
	passwordUpdater PasswordUpdater
	hashPassword    HashPasswordFunc
	verifyPassword  PasswordVerifyFunc
}

// NewPasswordService は PasswordService を生成する。
func NewPasswordService(
	userFinder UserFinderWithCredentials,
	historyStore PasswordHistoryStore,
	passwordUpdater PasswordUpdater,
	hashPassword HashPasswordFunc,
	verifyPassword PasswordVerifyFunc,
) *PasswordService {
	return &PasswordService{
		userFinder:      userFinder,
		historyStore:    historyStore,
		passwordUpdater: passwordUpdater,
		hashPassword:    hashPassword,
		verifyPassword:  verifyPassword,
	}
}

// ChangePassword はログイン済みユーザーのパスワードを変更する。
func (s *PasswordService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// ユーザー取得（クレデンシャル付き）
	user, err := s.userFinder.FindByIDWithCredentials(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrInvalidCredentials
	}

	// パスワードクレデンシャルを取得
	cred, hash := findPasswordCredential(user.Credentials)
	if cred == nil {
		return ErrInvalidCredentials
	}

	// 現在のパスワード検証
	match, err := s.verifyPassword(currentPassword, hash)
	if err != nil {
		return fmt.Errorf("failed to verify current password: %w", err)
	}
	if !match {
		return ErrInvalidCredentials
	}

	// 新パスワードが現在のパスワードと同じでないか確認
	sameAsCurrent, err := s.verifyPassword(newPassword, hash)
	if err != nil {
		return fmt.Errorf("failed to verify new password against current: %w", err)
	}
	if sameAsCurrent {
		return ErrPasswordSameAsCurrent
	}

	// パスワード履歴チェック（直近5件）
	histories, err := s.historyStore.FindRecentByUserID(ctx, userID, passwordHistoryLimit)
	if err != nil {
		return fmt.Errorf("failed to find password history: %w", err)
	}
	for _, h := range histories {
		match, err := s.verifyPassword(newPassword, h.PasswordHash)
		if err != nil {
			continue // ハッシュ解析エラーは無視して次へ
		}
		if match {
			return ErrPasswordInHistory
		}
	}

	// 新パスワードのハッシュ生成
	newHash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// パスワード更新
	if err := s.passwordUpdater.UpdateHash(ctx, cred.ID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// パスワード履歴に記録
	history := &model.PasswordHistory{
		UserID:       userID,
		PasswordHash: newHash,
	}
	if err := s.historyStore.Create(ctx, history); err != nil {
		return fmt.Errorf("failed to create password history: %w", err)
	}

	return nil
}

// findPasswordCredential はクレデンシャル一覧からパスワードタイプのクレデンシャルと
// そのハッシュを返す。見つからない場合は (nil, "") を返す。
func findPasswordCredential(credentials []model.Credential) (*model.Credential, string) {
	for i := range credentials {
		cred := &credentials[i]
		if cred.Type == "password" && cred.PasswordCredential != nil {
			return cred, cred.PasswordCredential.PasswordHash
		}
	}
	return nil, ""
}
