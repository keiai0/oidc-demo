package metrics

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// パスパラメータを正規化するパターン群。高カーディナリティを防ぐため、
// UUID・テナントコード・kid 等のダイナミックセグメントをプレースホルダに変換する。
var (
	uuidPattern       = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	dateKIDPattern    = regexp.MustCompile(`\d{4}-\d{2}-\d{2}-[0-9a-f]{8}`)
)

// normalizePath はパス中の動的セグメントをプレースホルダに置換する。
func normalizePath(path string) string {
	path = uuidPattern.ReplaceAllString(path, ":id")
	path = dateKIDPattern.ReplaceAllString(path, ":kid")
	return path
}

// HTTPMetricsMiddleware はリクエストごとにレイテンシとステータスコードを記録する。
func HTTPMetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := c.Response().Status
			path := normalizePath(c.Path())
			method := c.Request().Method

			HTTPRequestDuration.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Observe(duration)

			return err
		}
	}
}

// statusCode はエコーのレスポンスからステータスコードを文字列で返す。
func statusCode(status int) string {
	return strconv.Itoa(status)
}
