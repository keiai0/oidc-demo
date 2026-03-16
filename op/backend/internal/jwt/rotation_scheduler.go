package jwt

import (
	"context"
	"log/slog"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
)

// RotationScheduler はバックグラウンドで署名鍵のライフサイクル管理を行う。
//
// - rotationInterval を超過した active 鍵を passive に変更し新鍵を生成する
// - gracePeriod を超過した passive 鍵を expired に変更する
// - expired 鍵を DB から削除する
type RotationScheduler struct {
	keySvc           *KeyService
	auditLog         *audit.AuditLogger
	logger           *slog.Logger
	rotationInterval time.Duration // 例: 90日
	gracePeriod      time.Duration // 例: 7日
	checkInterval    time.Duration // チェック間隔（例: 1時間）
}

// NewRotationScheduler は RotationScheduler を生成する。
// rotationIntervalDays: ローテーション間隔（日）、gracePeriodDays: 猶予期間（日）
func NewRotationScheduler(
	keySvc *KeyService,
	auditLog *audit.AuditLogger,
	logger *slog.Logger,
	rotationIntervalDays int,
	gracePeriodDays int,
) *RotationScheduler {
	return &RotationScheduler{
		keySvc:           keySvc,
		auditLog:         auditLog,
		logger:           logger,
		rotationInterval: time.Duration(rotationIntervalDays) * 24 * time.Hour,
		gracePeriod:      time.Duration(gracePeriodDays) * 24 * time.Hour,
		checkInterval:    1 * time.Hour,
	}
}

// Run はバックグラウンドで鍵ライフサイクルを管理するループを起動する。
// ctx がキャンセルされると終了する。
func (s *RotationScheduler) Run(ctx context.Context) {
	s.logger.InfoContext(ctx, "key rotation scheduler started",
		"rotation_interval_days", int(s.rotationInterval.Hours()/24),
		"grace_period_days", int(s.gracePeriod.Hours()/24),
	)

	// 起動時に即一度チェックする
	s.tick(ctx)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "key rotation scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *RotationScheduler) tick(ctx context.Context) {
	if err := s.rotateIfNeeded(ctx); err != nil {
		s.logger.ErrorContext(ctx, "key rotation check failed", "error", err)
	}
	if err := s.keySvc.ExpirePassiveKeys(ctx, s.gracePeriod); err != nil {
		s.logger.ErrorContext(ctx, "passive key expiry failed", "error", err)
	}
	if err := s.keySvc.DeleteExpiredKeys(ctx); err != nil {
		s.logger.ErrorContext(ctx, "expired key deletion failed", "error", err)
	}
}

// rotateIfNeeded は active 鍵の作成日時が rotationInterval を超過していればローテーションを実行する。
func (s *RotationScheduler) rotateIfNeeded(ctx context.Context) error {
	activeKey, err := s.keySvc.signKeyRepo.FindActive(ctx)
	if err != nil {
		return err
	}
	if activeKey == nil {
		return nil
	}

	if time.Since(activeKey.CreatedAt) < s.rotationInterval {
		return nil
	}

	s.logger.InfoContext(ctx, "rotating signing key", "old_kid", activeKey.KID)

	newKey, err := s.keySvc.RotateKey(ctx)
	if err != nil {
		return err
	}

	s.auditLog.LogEvent(ctx, audit.EventKeyRotated,
		slog.String("old_kid", activeKey.KID),
		slog.String("new_kid", newKey.KID),
	)
	s.logger.InfoContext(ctx, "signing key rotated", "new_kid", newKey.KID)

	return nil
}
