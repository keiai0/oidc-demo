package audit

import (
	"context"
	"log/slog"
)

// AuditLogger は認証・認可イベントの監査ログを出力する。
// 構造化ログ（slog）で統一し、JSON 形式で出力する。
type AuditLogger struct {
	logger *slog.Logger
}

// New は AuditLogger を生成する。
func New(logger *slog.Logger) *AuditLogger {
	return &AuditLogger{logger: logger.With("component", "audit")}
}

// LogEvent は監査イベントを出力する。
func (a *AuditLogger) LogEvent(ctx context.Context, eventType string, attrs ...slog.Attr) {
	allAttrs := make([]slog.Attr, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.String("event_type", eventType))
	allAttrs = append(allAttrs, attrs...)

	a.logger.LogAttrs(ctx, slog.LevelInfo, eventType, allAttrs...)
}

// ヘルパー: よく使う属性を生成する関数群。

// UserAttr はユーザー ID 属性を返す。
func UserAttr(id string) slog.Attr {
	return slog.String("user_id", id)
}

// ClientAttr はクライアント ID 属性を返す。
func ClientAttr(id string) slog.Attr {
	return slog.String("client_id", id)
}

// IPAttr は IP アドレス属性を返す。
func IPAttr(ip string) slog.Attr {
	return slog.String("ip_address", ip)
}

// ResultAttr は結果属性を返す。
func ResultAttr(result string) slog.Attr {
	return slog.String("result", result)
}

// GrantTypeAttr は grant_type 属性を返す。
func GrantTypeAttr(grantType string) slog.Attr {
	return slog.String("grant_type", grantType)
}

// TenantAttr はテナントコード属性を返す。
func TenantAttr(tenantCode string) slog.Attr {
	return slog.String("tenant_code", tenantCode)
}

// MethodAttr は認証方法属性を返す。
func MethodAttr(method string) slog.Attr {
	return slog.String("method", method)
}
