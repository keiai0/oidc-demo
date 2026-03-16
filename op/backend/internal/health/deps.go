package health

import "context"

// DBPinger は DB 接続確認のインターフェース。
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// SigningKeyChecker は署名鍵のロード確認のインターフェース。
type SigningKeyChecker interface {
	HasActiveKey(ctx context.Context) bool
}
