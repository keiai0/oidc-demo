package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

const (
	// RFC 8693 Section 3: Token Type Identifiers
	TokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	TokenTypeIDToken     = "urn:ietf:params:oauth:token-type:id_token"
	TokenTypeJWT         = "urn:ietf:params:oauth:token-type:jwt"

	// grant_type 値
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
)

// TokenExchangeGrantInput は Token Exchange (RFC 8693) の入力。
type TokenExchangeGrantInput struct {
	ClientID           string
	ClientSecret       string
	SubjectToken       string
	SubjectTokenType   string
	ActorToken         string
	ActorTokenType     string
	Resource           string
	Audience           string
	Scope              string
	RequestedTokenType string
	DPoPJKT            string
	TenantCode         string
}

// TokenExchangeResponse は Token Exchange のレスポンス (RFC 8693 Section 2.2.1)。
// issued_token_type フィールドが REQUIRED なため TokenResponse とは別の型。
type TokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	Scope           string `json:"scope,omitempty"`
}

func (h *TokenHandler) handleTokenExchangeGrant(c echo.Context, dpopJKT string) error {
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	subjectToken := c.FormValue("subject_token")
	subjectTokenType := c.FormValue("subject_token_type")

	// subject_token と subject_token_type は REQUIRED (RFC 8693 Section 2.1)
	if subjectToken == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "subject_token is required")
	}
	if subjectTokenType == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "subject_token_type is required")
	}

	actorToken := c.FormValue("actor_token")
	actorTokenType := c.FormValue("actor_token_type")
	// actor_token_type は actor_token がある場合 REQUIRED
	if actorToken != "" && actorTokenType == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "actor_token_type is required when actor_token is present")
	}

	requestedTokenType := c.FormValue("requested_token_type")
	if requestedTokenType == "" {
		requestedTokenType = TokenTypeAccessToken
	}

	tenantCode := c.Param("tenant_code")

	resp, err := h.handleTokenExchangeGrantLogic(c.Request().Context(), &TokenExchangeGrantInput{
		ClientID:           clientID,
		ClientSecret:       clientSecret,
		SubjectToken:       subjectToken,
		SubjectTokenType:   subjectTokenType,
		ActorToken:         actorToken,
		ActorTokenType:     actorTokenType,
		Resource:           c.FormValue("resource"),
		Audience:           c.FormValue("audience"),
		Scope:              c.FormValue("scope"),
		RequestedTokenType: requestedTokenType,
		DPoPJKT:            dpopJKT,
		TenantCode:         tenantCode,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidClient) {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
		if errors.Is(err, ErrUnsupportedGrantType) {
			return tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "")
		}
		if errors.Is(err, ErrInvalidGrant) {
			return tokenError(c, http.StatusBadRequest, "invalid_grant", "")
		}
		if errors.Is(err, ErrInvalidTarget) {
			return tokenError(c, http.StatusBadRequest, "invalid_target", "")
		}
		if errors.Is(err, ErrInvalidScope) {
			return tokenError(c, http.StatusBadRequest, "invalid_scope", "")
		}
		h.logger.ErrorContext(c.Request().Context(), "token exchange error", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventTokenExchanged,
		audit.ClientAttr(clientID), audit.GrantTypeAttr(GrantTypeTokenExchange), audit.ResultAttr("success"),
	)
	metrics.TokenIssuedTotal.WithLabelValues("token-exchange").Inc()

	return c.JSON(http.StatusOK, resp)
}

// handleTokenExchangeGrantLogic は Token Exchange のビジネスロジック。
// 仕様参照: RFC 8693 Section 2
func (h *TokenHandler) handleTokenExchangeGrantLogic(ctx context.Context, input *TokenExchangeGrantInput) (*TokenExchangeResponse, error) {
	// 1. クライアント認証
	client, err := h.clientFinder.FindByClientID(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to find client: %w", err)
	}
	if client == nil || client.Status != "active" {
		return nil, ErrInvalidClient
	}

	match, err := h.verifyPassword(input.ClientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return nil, ErrInvalidClient
	}

	// 2. grant_type サポート確認
	if !client.HasGrantType(GrantTypeTokenExchange) {
		return nil, ErrUnsupportedGrantType
	}

	// 3. テナント解決
	tenant, err := h.tenantFinder.FindByCode(ctx, input.TenantCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidClient
	}

	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tenant-client association: %w", err)
	}
	if !belongs {
		return nil, ErrInvalidClient
	}

	// 4. ポリシー検索（ポリシーなし = token exchange 不許可）
	policy, err := h.tokenExchangePolicyFinder.FindByClientID(ctx, client.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find token exchange policy: %w", err)
	}
	if policy == nil {
		return nil, fmt.Errorf("no token exchange policy: %w", ErrInvalidGrant)
	}

	// 5. subject_token_type の許可チェック
	if !policy.HasSubjectTokenType(input.SubjectTokenType) {
		return nil, fmt.Errorf("subject_token_type not allowed: %w", ErrInvalidGrant)
	}

	// 6. requested_token_type の許可チェック
	if !policy.HasRequestedTokenType(input.RequestedTokenType) {
		return nil, fmt.Errorf("requested_token_type not allowed: %w", ErrInvalidGrant)
	}

	// 7. subject_token の検証
	subjectResult, err := h.validateSubjectToken(ctx, input.SubjectToken, input.SubjectTokenType)
	if err != nil {
		return nil, fmt.Errorf("invalid subject_token: %w", ErrInvalidGrant)
	}

	// 8. audience 検証
	audience := input.Audience
	if audience != "" && !policy.HasAudience(audience) {
		return nil, fmt.Errorf("audience not allowed: %w", ErrInvalidTarget)
	}
	if audience == "" {
		audience = client.ClientID
	}

	// 9. Impersonation vs Delegation の判定
	var actClaim *model.ActClaim
	if input.ActorToken != "" {
		// Delegation: actor_token あり
		if !policy.AllowDelegation {
			return nil, fmt.Errorf("delegation not allowed: %w", ErrInvalidGrant)
		}

		actorResult, err := h.validateActorToken(ctx, input.ActorToken, input.ActorTokenType)
		if err != nil {
			return nil, fmt.Errorf("invalid actor_token: %w", ErrInvalidGrant)
		}

		// may_act クレーム検証 (RFC 8693 Section 4.3)
		if err := h.validateMayAct(subjectResult, actorResult); err != nil {
			return nil, err
		}

		// act クレーム構築: 既存の act チェーンを維持 (RFC 8693 Section 4.1)
		actClaim = &model.ActClaim{
			Sub:      actorResult.Subject,
			ClientID: actorResult.ClientID,
		}
		// subject_token に既存の act がある場合はチェーンを維持
		if subjectResult.Act != nil {
			actClaim.Act = subjectResult.Act
		}
	} else {
		// Impersonation: actor_token なし
		if !policy.AllowImpersonation {
			return nil, fmt.Errorf("impersonation not allowed: %w", ErrInvalidGrant)
		}
	}

	// 10. スコープ制限（subject_token のスコープとの共通部分）
	resolvedScope := h.resolveExchangeScope(input.Scope, subjectResult.Scope)

	// 11. アクセストークン発行
	issuer := h.issuerBaseURL + "/" + tenant.Code
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second

	atClaims := &model.AccessTokenClaims{
		Issuer:   issuer,
		Subject:  subjectResult.Subject,
		Audience: audience,
		Scope:    resolvedScope,
		Act:      actClaim,
	}
	if input.DPoPJKT != "" {
		atClaims.Confirmation = &model.TokenConfirmation{JKT: input.DPoPJKT}
	}

	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, atClaims, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// DB 保存（SessionID = nil: Token Exchange は直接セッションに紐づかない）
	accessToken := &model.AccessToken{
		JTI:       accessJTI,
		SessionID: nil,
		ClientID:  client.ID,
		Scope:     resolvedScope,
		ExpiresAt: time.Now().Add(accessTokenLifetime),
	}
	if input.DPoPJKT != "" {
		accessToken.DPoPJKT = &input.DPoPJKT
	}
	if err := h.accessTokenStore.Create(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to save access token: %w", err)
	}

	tokenType := "Bearer"
	if input.DPoPJKT != "" {
		tokenType = "DPoP"
	}

	return &TokenExchangeResponse{
		AccessToken:     accessTokenStr,
		IssuedTokenType: input.RequestedTokenType,
		TokenType:       tokenType,
		ExpiresIn:       tenant.AccessTokenLifetime,
		Scope:           resolvedScope,
	}, nil
}

// validateSubjectToken は subject_token を token_type に応じて検証する。
func (h *TokenHandler) validateSubjectToken(ctx context.Context, token, tokenType string) (*model.AccessTokenResult, error) {
	switch tokenType {
	case TokenTypeAccessToken:
		return h.validateAccessTokenWithRevocationCheck(ctx, token)
	case TokenTypeJWT:
		// JWT タイプもアクセストークンとして検証（このOPが発行したJWTのみサポート）
		return h.validateAccessTokenWithRevocationCheck(ctx, token)
	default:
		return nil, fmt.Errorf("unsupported subject_token_type: %s", tokenType)
	}
}

// validateActorToken は actor_token を検証する。
func (h *TokenHandler) validateActorToken(ctx context.Context, token, tokenType string) (*model.AccessTokenResult, error) {
	switch tokenType {
	case TokenTypeAccessToken:
		return h.validateAccessTokenWithRevocationCheck(ctx, token)
	case TokenTypeJWT:
		return h.validateAccessTokenWithRevocationCheck(ctx, token)
	default:
		return nil, fmt.Errorf("unsupported actor_token_type: %s", tokenType)
	}
}

// validateAccessTokenWithRevocationCheck は JWT 署名検証 + DB での失効チェックを行う。
func (h *TokenHandler) validateAccessTokenWithRevocationCheck(ctx context.Context, tokenStr string) (*model.AccessTokenResult, error) {
	result, err := h.tokenValidator.ValidateAccessToken(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// DB で失効・期限切れをチェック
	dbToken, err := h.accessTokenStore.FindByJTI(ctx, result.JTI)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup token: %w", err)
	}
	if dbToken == nil {
		return nil, fmt.Errorf("token not found in database")
	}
	if dbToken.RevokedAt != nil {
		return nil, fmt.Errorf("token has been revoked")
	}
	if dbToken.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	return result, nil
}

// validateMayAct は subject_token の may_act クレームを検証する (RFC 8693 Section 4.3)。
// may_act が存在しない場合は制限なしとして許可する。
func (h *TokenHandler) validateMayAct(subjectResult, actorResult *model.AccessTokenResult) error {
	// 現在の実装では may_act クレームをアクセストークンに含めていないため、
	// 常に許可する。将来 may_act をサポートする場合はここで検証を追加する。
	return nil
}

// resolveExchangeScope は要求されたスコープと subject_token のスコープの共通部分を返す。
// 要求が空の場合は subject_token のスコープをそのまま使う。
func (h *TokenHandler) resolveExchangeScope(requestedScope, subjectScope string) string {
	if requestedScope == "" {
		return subjectScope
	}

	subjectScopes := strings.Split(subjectScope, " ")
	subjectSet := make(map[string]bool, len(subjectScopes))
	for _, s := range subjectScopes {
		if s != "" {
			subjectSet[s] = true
		}
	}

	requestedScopes := strings.Split(requestedScope, " ")
	var result []string
	for _, s := range requestedScopes {
		if s != "" && subjectSet[s] {
			result = append(result, s)
		}
	}

	return strings.Join(result, " ")
}
