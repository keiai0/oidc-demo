package scim

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SCIMError は RFC 7644 Section 3.12 に準拠した SCIM エラーレスポンス。
type SCIMError struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

func scimError(c echo.Context, httpStatus int, detail, scimType string) error {
	return c.JSON(httpStatus, SCIMError{
		Schemas:  []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Detail:   detail,
		Status:   http.StatusText(httpStatus),
		ScimType: scimType,
	})
}

func scimErrorNotFound(c echo.Context) error {
	return scimError(c, http.StatusNotFound, "Resource not found", "")
}

func scimErrorBadRequest(c echo.Context, detail string) error {
	return scimError(c, http.StatusBadRequest, detail, "invalidValue")
}

func scimErrorConflict(c echo.Context, detail string) error {
	return scimError(c, http.StatusConflict, detail, "uniqueness")
}

func scimErrorServer(c echo.Context) error {
	return scimError(c, http.StatusInternalServerError, "Internal server error", "")
}
