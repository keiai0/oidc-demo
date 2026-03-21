package scim

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SchemaHandler は SCIM スキーマ関連のエンドポイントを処理する。
type SchemaHandler struct{}

// NewSchemaHandler は SchemaHandler を生成する。
func NewSchemaHandler() *SchemaHandler {
	return &SchemaHandler{}
}

// HandleSchemas は GET /{tenant_code}/scim/v2/Schemas を処理する。
// RFC 7643 Section 8.7
func (h *SchemaHandler) HandleSchemas(c echo.Context) error {
	schemas := []map[string]interface{}{
		{
			"id":          "urn:ietf:params:scim:schemas:core:2.0:User",
			"name":        "User",
			"description": "User Account",
			"attributes": []map[string]interface{}{
				{"name": "userName", "type": "string", "required": true, "uniqueness": "server"},
				{"name": "name", "type": "complex", "subAttributes": []map[string]interface{}{
					{"name": "formatted", "type": "string"},
				}},
				{"name": "emails", "type": "complex", "multiValued": true, "subAttributes": []map[string]interface{}{
					{"name": "value", "type": "string"},
					{"name": "primary", "type": "boolean"},
				}},
				{"name": "active", "type": "boolean"},
				{"name": "externalId", "type": "string"},
			},
			"meta": map[string]string{"resourceType": "Schema", "location": "/Schemas/urn:ietf:params:scim:schemas:core:2.0:User"},
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(schemas),
		"Resources":    schemas,
	})
}

// HandleServiceProviderConfig は GET /{tenant_code}/scim/v2/ServiceProviderConfig を処理する。
// RFC 7643 Section 5
func (h *SchemaHandler) HandleServiceProviderConfig(c echo.Context) error {
	config := map[string]interface{}{
		"schemas":       []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":         map[string]bool{"supported": true},
		"bulk":          map[string]interface{}{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":        map[string]interface{}{"supported": true, "maxResults": 100},
		"changePassword": map[string]bool{"supported": false},
		"sort":          map[string]bool{"supported": false},
		"etag":          map[string]bool{"supported": false},
		"authenticationSchemes": []map[string]string{
			{
				"type":        "oauthbearertoken",
				"name":        "OAuth Bearer Token",
				"description": "Authentication scheme using the OAuth Bearer Token Standard",
			},
		},
	}

	return c.JSON(http.StatusOK, config)
}

// HandleResourceTypes は GET /{tenant_code}/scim/v2/ResourceTypes を処理する。
// RFC 7643 Section 6
func (h *SchemaHandler) HandleResourceTypes(c echo.Context) error {
	resourceTypes := []map[string]interface{}{
		{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":          "User",
			"name":        "User",
			"endpoint":    "/Users",
			"schema":      "urn:ietf:params:scim:schemas:core:2.0:User",
			"meta":        map[string]string{"resourceType": "ResourceType", "location": "/ResourceTypes/User"},
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(resourceTypes),
		"Resources":    resourceTypes,
	})
}
