package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LoginTotal はログイン試行数。result: "success" / "failure" / "locked"
	LoginTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "op_login_total",
		Help: "Total number of login attempts",
	}, []string{"result"})

	// TokenIssuedTotal はトークン発行数。grant_type: "authorization_code" / "refresh_token"
	TokenIssuedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "op_token_issued_total",
		Help: "Total number of tokens issued",
	}, []string{"grant_type"})

	// TokenRevokedTotal はトークン失効数。
	TokenRevokedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "op_token_revoked_total",
		Help: "Total number of tokens revoked",
	})

	// ActiveSessions は現在のアクティブセッション数。
	ActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "op_active_sessions",
		Help: "Current number of active sessions",
	})

	// HTTPRequestDuration はエンドポイント別レスポンスタイム。
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "op_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)
