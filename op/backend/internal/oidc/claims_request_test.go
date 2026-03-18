package oidc

import (
	"testing"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

func Test_ParseClaimsRequest(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "empty string returns nil",
			raw:     "",
			wantNil: true,
		},
		{
			name:    "valid id_token claims",
			raw:     `{"id_token":{"email":{"essential":true}}}`,
			wantNil: false,
		},
		{
			name:    "valid userinfo claims",
			raw:     `{"userinfo":{"name":null}}`,
			wantNil: false,
		},
		{
			name:    "invalid JSON",
			raw:     `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseClaimsRequest(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseClaimsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && got != nil {
				t.Errorf("ParseClaimsRequest() = %v, want nil", got)
			}
			if !tt.wantNil && !tt.wantErr && got == nil {
				t.Error("ParseClaimsRequest() = nil, want non-nil")
			}
		})
	}
}

func Test_ResolveIDTokenClaims(t *testing.T) {
	userName := "Test User"
	user := &model.User{
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          &userName,
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	session := &model.Session{
		AuthTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		ACR:      "urn:mace:incommon:iap:bronze",
		AMR:      model.StringSlice{"pwd"},
	}

	tests := []struct {
		name       string
		cr         *model.ClaimsRequest
		wantClaims []string
	}{
		{
			name: "email claim",
			cr: &model.ClaimsRequest{
				IDToken: map[string]*model.ClaimSpec{
					"email": {Essential: true},
				},
			},
			wantClaims: []string{"email"},
		},
		{
			name: "session claims",
			cr: &model.ClaimsRequest{
				IDToken: map[string]*model.ClaimSpec{
					"auth_time": nil,
					"acr":       nil,
				},
			},
			wantClaims: []string{"auth_time", "acr"},
		},
		{
			name: "unsupported claim ignored",
			cr: &model.ClaimsRequest{
				IDToken: map[string]*model.ClaimSpec{
					"unknown_claim": nil,
				},
			},
			wantClaims: nil,
		},
		{
			name:       "nil claims request",
			cr:         nil,
			wantClaims: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveIDTokenClaims(tt.cr, user, session)
			if tt.wantClaims == nil {
				if got != nil {
					t.Errorf("ResolveIDTokenClaims() = %v, want nil", got)
				}
				return
			}
			for _, claim := range tt.wantClaims {
				if _, ok := got[claim]; !ok {
					t.Errorf("ResolveIDTokenClaims() missing claim %q", claim)
				}
			}
		})
	}
}

func Test_ResolveUserinfoClaims(t *testing.T) {
	userName := "Test User"
	user := &model.User{
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          &userName,
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name       string
		cr         *model.ClaimsRequest
		wantClaims []string
	}{
		{
			name: "name and email claims",
			cr: &model.ClaimsRequest{
				Userinfo: map[string]*model.ClaimSpec{
					"name":  nil,
					"email": {Essential: true},
				},
			},
			wantClaims: []string{"name", "email"},
		},
		{
			name:       "nil claims request",
			cr:         nil,
			wantClaims: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveUserinfoClaims(tt.cr, user)
			if tt.wantClaims == nil {
				if got != nil {
					t.Errorf("ResolveUserinfoClaims() = %v, want nil", got)
				}
				return
			}
			for _, claim := range tt.wantClaims {
				if _, ok := got[claim]; !ok {
					t.Errorf("ResolveUserinfoClaims() missing claim %q", claim)
				}
			}
		})
	}
}
