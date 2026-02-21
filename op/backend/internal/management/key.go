package management

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// KeyHandler は署名鍵管理エンドポイントを処理する。
type KeyHandler struct {
	signKeyStore SignKeyStore
	keyRotator   KeyRotator
}

// NewKeyHandler は KeyHandler を生成する。
func NewKeyHandler(signKeyStore SignKeyStore, keyRotator KeyRotator) *KeyHandler {
	return &KeyHandler{
		signKeyStore: signKeyStore,
		keyRotator:   keyRotator,
	}
}

type keyResponse struct {
	KID       string  `json:"kid"`
	Algorithm string  `json:"algorithm"`
	Active    bool    `json:"active"`
	CreatedAt string  `json:"created_at"`
	RotatedAt *string `json:"rotated_at,omitempty"`
}

// HandleList は GET /management/v1/keys を処理する。
func (h *KeyHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()

	keys, err := h.signKeyStore.FindAll(ctx)
	if err != nil {
		c.Logger().Errorf("failed to list keys: %v", err)
		return serverError(c)
	}

	data := make([]keyResponse, len(keys))
	for i, k := range keys {
		resp := keyResponse{
			KID:       k.KID,
			Algorithm: k.Algorithm,
			Active:    k.Active,
			CreatedAt: k.CreatedAt.Format(time.RFC3339),
		}
		if k.RotatedAt != nil {
			s := k.RotatedAt.Format(time.RFC3339)
			resp.RotatedAt = &s
		}
		data[i] = resp
	}

	return c.JSON(http.StatusOK, data)
}

// HandleRotate は POST /management/v1/keys/rotate を処理する。
func (h *KeyHandler) HandleRotate(c echo.Context) error {
	ctx := c.Request().Context()

	newKey, err := h.keyRotator.RotateKey(ctx)
	if err != nil {
		c.Logger().Errorf("failed to rotate key: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, keyResponse{
		KID:       newKey.KID,
		Algorithm: newKey.Algorithm,
		Active:    newKey.Active,
		CreatedAt: newKey.CreatedAt.Format(time.RFC3339),
	})
}

// HandleDeactivate は DELETE /management/v1/keys/:kid を処理する。
func (h *KeyHandler) HandleDeactivate(c echo.Context) error {
	ctx := c.Request().Context()
	kid := c.Param("kid")

	key, err := h.signKeyStore.FindByKID(ctx, kid)
	if err != nil {
		c.Logger().Errorf("failed to find key: %v", err)
		return serverError(c)
	}
	if key == nil {
		return notFound(c, "key not found")
	}
	if !key.Active {
		return badRequest(c, "key is already inactive")
	}

	// 最後のアクティブ鍵の無効化を防ぐ
	activeKeys, err := h.signKeyStore.FindAllActive(ctx)
	if err != nil {
		c.Logger().Errorf("failed to check active keys: %v", err)
		return serverError(c)
	}
	if len(activeKeys) <= 1 {
		return badRequest(c, "cannot deactivate the last active key")
	}

	if err := h.signKeyStore.Deactivate(ctx, kid); err != nil {
		c.Logger().Errorf("failed to deactivate key: %v", err)
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}
