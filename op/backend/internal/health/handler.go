package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// LivenessHandler は GET /health を処理する。
// アプリケーションが生存しているかを確認するエンドポイント。DB チェックは行わない。
type LivenessHandler struct{}

// NewLivenessHandler は LivenessHandler を生成する。
func NewLivenessHandler() *LivenessHandler {
	return &LivenessHandler{}
}

// Handle は常に 200 OK を返す。
func (h *LivenessHandler) Handle(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ReadinessHandler は GET /ready を処理する。
// DB 接続と署名鍵のロードを確認し、サービスが要求を処理できる状態かを返す。
type ReadinessHandler struct {
	dbPinger    DBPinger
	keyChecker  SigningKeyChecker
}

// NewReadinessHandler は ReadinessHandler を生成する。
func NewReadinessHandler(dbPinger DBPinger, keyChecker SigningKeyChecker) *ReadinessHandler {
	return &ReadinessHandler{dbPinger: dbPinger, keyChecker: keyChecker}
}

// Handle は DB ping と署名鍵確認を実行し、準備完了なら 200、異常なら 503 を返す。
func (h *ReadinessHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	checks := map[string]string{}
	healthy := true

	if err := h.dbPinger.PingContext(ctx); err != nil {
		checks["db"] = "unavailable"
		healthy = false
	} else {
		checks["db"] = "ok"
	}

	if !h.keyChecker.HasActiveKey(ctx) {
		checks["signing_key"] = "not_loaded"
		healthy = false
	} else {
		checks["signing_key"] = "ok"
	}

	if !healthy {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"checks": checks,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ready",
		"checks": checks,
	})
}
