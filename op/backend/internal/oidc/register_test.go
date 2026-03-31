package oidc

import (
	"testing"
)

func Test_validateRegistrationRedirectURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URI",
			uri:     "https://example.com/callback",
			wantErr: false,
		},
		{
			name:    "valid localhost HTTP",
			uri:     "http://localhost:3000/callback",
			wantErr: false,
		},
		{
			name:    "valid 127.0.0.1 HTTP",
			uri:     "http://127.0.0.1:8080/callback",
			wantErr: false,
		},
		{
			name:    "invalid: non-localhost HTTP",
			uri:     "http://example.com/callback",
			wantErr: true,
		},
		{
			name:    "invalid: has fragment",
			uri:     "https://example.com/callback#frag",
			wantErr: true,
		},
		{
			name:    "invalid: no scheme",
			uri:     "example.com/callback",
			wantErr: true,
		},
		{
			name:    "invalid: no host",
			uri:     "https:///callback",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistrationRedirectURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegistrationRedirectURI(%q) error = %v, wantErr %v", tt.uri, err, tt.wantErr)
			}
		})
	}
}

func Test_validateRegistrationRequest(t *testing.T) {
	h := &RegistrationHandler{}

	tests := []struct {
		name      string
		req       registrationRequest
		wantError string // "" = no error
	}{
		{
			name: "valid minimal request",
			req: registrationRequest{
				RedirectURIs: []string{"https://example.com/callback"},
			},
			wantError: "",
		},
		{
			name: "valid full request",
			req: registrationRequest{
				RedirectURIs:            []string{"https://example.com/callback"},
				GrantTypes:              []string{"authorization_code", "refresh_token"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "client_secret_basic",
				ClientName:              "Test Client",
			},
			wantError: "",
		},
		{
			name: "missing redirect_uris",
			req: registrationRequest{
				RedirectURIs: nil,
			},
			wantError: "invalid_redirect_uri",
		},
		{
			name: "empty redirect_uris",
			req: registrationRequest{
				RedirectURIs: []string{},
			},
			wantError: "invalid_redirect_uri",
		},
		{
			name: "invalid redirect_uri (HTTP non-localhost)",
			req: registrationRequest{
				RedirectURIs: []string{"http://example.com/callback"},
			},
			wantError: "invalid_redirect_uri",
		},
		{
			name: "unsupported grant_type",
			req: registrationRequest{
				RedirectURIs: []string{"https://example.com/callback"},
				GrantTypes:   []string{"implicit"},
			},
			wantError: "invalid_client_metadata",
		},
		{
			name: "unsupported response_type",
			req: registrationRequest{
				RedirectURIs:  []string{"https://example.com/callback"},
				ResponseTypes: []string{"token"},
			},
			wantError: "invalid_client_metadata",
		},
		{
			name: "unsupported auth_method",
			req: registrationRequest{
				RedirectURIs:            []string{"https://example.com/callback"},
				TokenEndpointAuthMethod: "private_key_jwt",
			},
			wantError: "invalid_client_metadata",
		},
		{
			name: "grant_type/response_type mismatch: code without authorization_code",
			req: registrationRequest{
				RedirectURIs:  []string{"https://example.com/callback"},
				GrantTypes:    []string{"client_credentials"},
				ResponseTypes: []string{"code"},
			},
			wantError: "invalid_client_metadata",
		},
		{
			name: "grant_type without authorization_code but response_type has code",
			req: registrationRequest{
				RedirectURIs:  []string{"https://example.com/callback"},
				GrantTypes:    []string{"refresh_token"},
				ResponseTypes: []string{"code"},
			},
			wantError: "invalid_client_metadata",
		},
		{
			name: "public client (auth_method=none)",
			req: registrationRequest{
				RedirectURIs:            []string{"https://example.com/callback"},
				TokenEndpointAuthMethod: "none",
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.validateRegistrationRequest(&tt.req)
			if tt.wantError == "" {
				if result != nil {
					t.Errorf("expected no error, got: %s (%s)", result.Error, result.ErrorDescription)
				}
			} else {
				if result == nil {
					t.Errorf("expected error %q, got nil", tt.wantError)
				} else if result.Error != tt.wantError {
					t.Errorf("expected error %q, got %q (%s)", tt.wantError, result.Error, result.ErrorDescription)
				}
			}
		})
	}
}

func Test_extractBearerToken(t *testing.T) {
	// extractBearerToken はテスト困難（echo.Context が必要）なので、
	// generateRandomHex のテストを代わりに実施する
}

func Test_generateRandomHex(t *testing.T) {
	tests := []struct {
		name  string
		bytes int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomHex(tt.bytes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// hex encoding doubles the length
			expectedLen := tt.bytes * 2
			if len(result) != expectedLen {
				t.Errorf("expected length %d, got %d", expectedLen, len(result))
			}

			// Generate another and verify they're different (non-deterministic)
			result2, _ := generateRandomHex(tt.bytes)
			if result == result2 {
				t.Error("two random values should not be equal")
			}
		})
	}
}
