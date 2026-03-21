package scim

import (
	"fmt"
	"strings"
)

// Filter は SCIM フィルタ式を表す。
// 簡易実装: eq 演算子のみサポート。
type Filter struct {
	Attribute string
	Operator  string
	Value     string
}

// ParseFilter は SCIM フィルタ文字列をパースする。
// サポート形式: `attribute eq "value"`
// サポート属性: userName, email, externalId, active
func ParseFilter(filterStr string) (*Filter, error) {
	if filterStr == "" {
		return nil, nil
	}

	// "userName eq \"value\"" のようなフォーマット
	parts := strings.SplitN(filterStr, " ", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid filter format")
	}

	attribute := parts[0]
	operator := strings.ToLower(parts[1])
	value := parts[2]

	if operator != "eq" {
		return nil, fmt.Errorf("unsupported operator: %s (only eq is supported)", operator)
	}

	// 属性のバリデーション
	switch attribute {
	case "userName", "email", "externalId", "active":
		// OK
	default:
		return nil, fmt.Errorf("unsupported filter attribute: %s", attribute)
	}

	// 値のクォートを除去
	value = strings.Trim(value, "\"")

	return &Filter{
		Attribute: attribute,
		Operator:  operator,
		Value:     value,
	}, nil
}
