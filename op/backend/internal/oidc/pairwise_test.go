package oidc

import (
	"testing"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

func Test_ComputePairwiseSub(t *testing.T) {
	tests := []struct {
		name             string
		sectorIdentifier string
		userID           string
		salt             string
		wantConsistent   bool // 同じ入力で同じ出力か
	}{
		{
			name:             "deterministic",
			sectorIdentifier: "example.com",
			userID:           "user-123",
			salt:             "random-salt",
			wantConsistent:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub1 := ComputePairwiseSub(tt.sectorIdentifier, tt.userID, tt.salt)
			sub2 := ComputePairwiseSub(tt.sectorIdentifier, tt.userID, tt.salt)

			if sub1 == "" {
				t.Error("ComputePairwiseSub returned empty string")
			}
			if tt.wantConsistent && sub1 != sub2 {
				t.Errorf("ComputePairwiseSub not deterministic: %s != %s", sub1, sub2)
			}
		})
	}
}

func Test_ComputePairwiseSub_DifferentInputs(t *testing.T) {
	salt := "test-salt"

	// 異なるユーザー → 異なる sub
	sub1 := ComputePairwiseSub("example.com", "user-1", salt)
	sub2 := ComputePairwiseSub("example.com", "user-2", salt)
	if sub1 == sub2 {
		t.Error("different users should produce different subs")
	}

	// 異なる sector → 異なる sub
	sub3 := ComputePairwiseSub("example.com", "user-1", salt)
	sub4 := ComputePairwiseSub("other.com", "user-1", salt)
	if sub3 == sub4 {
		t.Error("different sectors should produce different subs")
	}

	// 異なる salt → 異なる sub
	sub5 := ComputePairwiseSub("example.com", "user-1", "salt-a")
	sub6 := ComputePairwiseSub("example.com", "user-1", "salt-b")
	if sub5 == sub6 {
		t.Error("different salts should produce different subs")
	}
}

func Test_GetSectorIdentifier(t *testing.T) {
	sectorURI := "https://rp.example.com/sector"

	tests := []struct {
		name   string
		client *model.Client
		want   string
	}{
		{
			name: "from sector_identifier_uri",
			client: &model.Client{
				SectorIdentifierURI: &sectorURI,
				RedirectURIs: []model.RedirectURI{
					{URI: "https://other.example.com/callback"},
				},
			},
			want: "rp.example.com",
		},
		{
			name: "fallback to redirect_uri host",
			client: &model.Client{
				RedirectURIs: []model.RedirectURI{
					{URI: "https://app.example.com/callback"},
				},
			},
			want: "app.example.com",
		},
		{
			name:   "no URIs returns empty",
			client: &model.Client{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSectorIdentifier(tt.client)
			if got != tt.want {
				t.Errorf("GetSectorIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_ResolveSubject(t *testing.T) {
	salt := "test-salt"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name   string
		client *model.Client
		wantRaw bool // true なら userID がそのまま返される
	}{
		{
			name:    "public subject type",
			client:  &model.Client{SubjectType: "public"},
			wantRaw: true,
		},
		{
			name: "pairwise subject type",
			client: &model.Client{
				SubjectType:  "pairwise",
				PairwiseSalt: &salt,
				RedirectURIs: []model.RedirectURI{
					{URI: "https://example.com/callback"},
				},
			},
			wantRaw: false,
		},
		{
			name:    "pairwise without salt falls back to raw",
			client:  &model.Client{SubjectType: "pairwise"},
			wantRaw: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSubject(tt.client, userID)
			if tt.wantRaw {
				if got != userID {
					t.Errorf("ResolveSubject() = %q, want raw userID %q", got, userID)
				}
			} else {
				if got == userID {
					t.Error("ResolveSubject() returned raw userID for pairwise client")
				}
				if got == "" {
					t.Error("ResolveSubject() returned empty string")
				}
			}
		})
	}
}
