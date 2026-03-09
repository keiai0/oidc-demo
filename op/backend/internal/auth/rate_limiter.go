package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type ipEntry struct {
	count    int
	windowAt time.Time
}

// RateLimiter は IP ベースのスライディングウィンドウ方式レート制限。
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*ipEntry
	limit    int
	window   time.Duration
	nowFunc  func() time.Time
}

// NewRateLimiter は指定した制限値とウィンドウでレートリミッターを生成する。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*ipEntry),
		limit:   limit,
		window:  window,
		nowFunc: time.Now,
	}
}

// Middleware は Echo ミドルウェアとしてレート制限を適用する。
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !rl.allow(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error":             "too_many_requests",
					"error_description": "rate limit exceeded, please try again later",
				})
			}
			return next(c)
		}
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFunc()
	entry, exists := rl.entries[ip]

	if !exists || now.Sub(entry.windowAt) > rl.window {
		rl.entries[ip] = &ipEntry{count: 1, windowAt: now}
		return true
	}

	entry.count++
	return entry.count <= rl.limit
}
