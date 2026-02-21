package management

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// ListResponse はページネーション付きレスポンスのラッパー。
type ListResponse[T any] struct {
	Data       []T   `json:"data"`
	TotalCount int64 `json:"total_count"`
}

// PaginationParams はパース済みのページネーションクエリパラメータを保持する。
type PaginationParams struct {
	Limit  int
	Offset int
}

const (
	defaultLimit = 50
	maxLimit     = 100
)

// parsePagination はクエリパラメータから limit/offset を抽出し、デフォルト値を適用する。
func parsePagination(c echo.Context) PaginationParams {
	limit := defaultLimit
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := 0
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return PaginationParams{Limit: limit, Offset: offset}
}
