package scim

import "testing"

func Test_ParseFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantErr   bool
		wantAttr  string
		wantValue string
	}{
		{name: "empty string returns nil", input: "", wantNil: true},
		{name: "valid userName eq", input: `userName eq "testuser"`, wantAttr: "userName", wantValue: "testuser"},
		{name: "valid email eq", input: `email eq "test@example.com"`, wantAttr: "email", wantValue: "test@example.com"},
		{name: "valid externalId eq", input: `externalId eq "ext-123"`, wantAttr: "externalId", wantValue: "ext-123"},
		{name: "valid active eq", input: `active eq "true"`, wantAttr: "active", wantValue: "true"},
		{name: "unsupported operator", input: `userName co "test"`, wantErr: true},
		{name: "unsupported attribute", input: `displayName eq "test"`, wantErr: true},
		{name: "malformed filter", input: `userName`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseFilter() = %v, want nil", got)
				}
				return
			}
			if tt.wantErr {
				return
			}
			if got.Attribute != tt.wantAttr {
				t.Errorf("Attribute = %q, want %q", got.Attribute, tt.wantAttr)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
		})
	}
}
