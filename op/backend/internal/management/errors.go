package management

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorResponse は管理 API の標準エラーレスポンス形式。
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func errorJSON(c echo.Context, status int, errCode, description string) error {
	return c.JSON(status, ErrorResponse{
		Error:            errCode,
		ErrorDescription: description,
	})
}

func badRequest(c echo.Context, description string) error {
	return errorJSON(c, http.StatusBadRequest, "bad_request", description)
}

func notFound(c echo.Context, description string) error {
	return errorJSON(c, http.StatusNotFound, "not_found", description)
}

func conflict(c echo.Context, description string) error {
	return errorJSON(c, http.StatusConflict, "conflict", description)
}

func serverError(c echo.Context) error {
	return errorJSON(c, http.StatusInternalServerError, "server_error", "internal server error")
}
