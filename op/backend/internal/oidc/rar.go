package oidc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// ParseAuthorizationDetails は authorization_details パラメータの JSON 文字列をパースする。
// 仕様参照: RFC 9396 Section 2
func ParseAuthorizationDetails(rawJSON string) ([]model.AuthorizationDetail, error) {
	if rawJSON == "" {
		return nil, nil
	}

	// authorization_details は JSON 配列
	var rawDetails []json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &rawDetails); err != nil {
		return nil, fmt.Errorf("authorization_details must be a JSON array: %w", err)
	}

	if len(rawDetails) == 0 {
		return nil, fmt.Errorf("authorization_details must not be empty")
	}

	details := make([]model.AuthorizationDetail, 0, len(rawDetails))
	for i, raw := range rawDetails {
		detail, err := parseOneDetail(raw)
		if err != nil {
			return nil, fmt.Errorf("authorization_details[%d]: %w", i, err)
		}
		details = append(details, *detail)
	}

	return details, nil
}

// parseOneDetail は authorization_details 配列の 1 要素をパースする。
// 共通フィールド (RFC 9396 Section 2) を構造体に、
// タイプ固有フィールドは Extra マップに格納する。
func parseOneDetail(raw json.RawMessage) (*model.AuthorizationDetail, error) {
	// まず全体を map として読み込む
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("must be a JSON object: %w", err)
	}

	// type フィールドは REQUIRED (RFC 9396 Section 2)
	typeName, ok := m["type"].(string)
	if !ok || typeName == "" {
		return nil, fmt.Errorf("\"type\" field is required and must be a non-empty string")
	}

	detail := &model.AuthorizationDetail{
		Type:  typeName,
		Extra: make(map[string]interface{}),
	}

	// 共通フィールドの抽出
	if v, ok := m["locations"]; ok {
		detail.Locations = toStringSlice(v)
	}
	if v, ok := m["actions"]; ok {
		detail.Actions = toStringSlice(v)
	}
	if v, ok := m["datatypes"]; ok {
		detail.DataTypes = toStringSlice(v)
	}
	if v, ok := m["identifier"].(string); ok {
		detail.Identifier = v
	}
	if v, ok := m["privileges"]; ok {
		detail.Privileges = toStringSlice(v)
	}

	// タイプ固有フィールドを Extra に格納
	commonFields := map[string]bool{
		"type": true, "locations": true, "actions": true,
		"datatypes": true, "identifier": true, "privileges": true,
	}
	for k, v := range m {
		if !commonFields[k] {
			detail.Extra[k] = v
		}
	}

	return detail, nil
}

// ValidateAuthorizationDetails は authorization_details の各要素を検証する。
// テナントでサポートされている type かどうか、allowed_actions/locations の範囲内かを確認する。
func ValidateAuthorizationDetails(
	ctx context.Context,
	details []model.AuthorizationDetail,
	tenantID uuid.UUID,
	typeFinder AuthorizationDetailTypeFinder,
) error {
	for i, detail := range details {
		adt, err := typeFinder.FindByTenantIDAndType(ctx, tenantID, detail.Type)
		if err != nil {
			return fmt.Errorf("authorization_details[%d]: failed to look up type: %w", i, err)
		}
		if adt == nil {
			return fmt.Errorf("authorization_details[%d]: unsupported type %q", i, detail.Type)
		}

		// actions の許可チェック
		if len(adt.AllowedActions) > 0 && len(detail.Actions) > 0 {
			allowed := toSet([]string(adt.AllowedActions))
			for _, action := range detail.Actions {
				if !allowed[action] {
					return fmt.Errorf("authorization_details[%d]: action %q is not allowed for type %q", i, action, detail.Type)
				}
			}
		}

		// locations の許可チェック
		if len(adt.AllowedLocations) > 0 && len(detail.Locations) > 0 {
			allowed := toSet([]string(adt.AllowedLocations))
			for _, location := range detail.Locations {
				if !allowed[location] {
					return fmt.Errorf("authorization_details[%d]: location %q is not allowed for type %q", i, location, detail.Type)
				}
			}
		}
	}

	return nil
}

// SerializeAuthorizationDetails は AuthorizationDetail スライスを JSON 文字列に変換する。
// 共通フィールドとタイプ固有フィールド (Extra) をマージして出力する。
func SerializeAuthorizationDetails(details []model.AuthorizationDetail) (string, error) {
	if len(details) == 0 {
		return "", nil
	}

	result := make([]map[string]interface{}, 0, len(details))
	for _, d := range details {
		m := make(map[string]interface{})
		m["type"] = d.Type
		if len(d.Locations) > 0 {
			m["locations"] = d.Locations
		}
		if len(d.Actions) > 0 {
			m["actions"] = d.Actions
		}
		if len(d.DataTypes) > 0 {
			m["datatypes"] = d.DataTypes
		}
		if d.Identifier != "" {
			m["identifier"] = d.Identifier
		}
		if len(d.Privileges) > 0 {
			m["privileges"] = d.Privileges
		}
		// タイプ固有フィールド
		for k, v := range d.Extra {
			m[k] = v
		}
		result = append(result, m)
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to serialize authorization_details: %w", err)
	}
	return string(bytes), nil
}

// toStringSlice は interface{} を []string に変換する。
func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}
