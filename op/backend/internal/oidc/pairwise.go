package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// ComputePairwiseSub は Pairwise Subject Identifier を計算する。
// 仕様参照: OIDC Core 1.0 Section 8.1
// sub = base64url(SHA-256(sector_identifier + user_id + salt))
func ComputePairwiseSub(sectorIdentifier, userID, salt string) string {
	input := sectorIdentifier + userID + salt
	hash := sha256.Sum256([]byte(input))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GetSectorIdentifier はクライアントの sector identifier を返す。
// SectorIdentifierURI が設定されている場合はそのホスト部を使用し、
// そうでなければ最初の redirect_uri のホスト部を使用する。
// 仕様参照: OIDC Core 1.0 Section 8.1
func GetSectorIdentifier(client *model.Client) string {
	if client.SectorIdentifierURI != nil && *client.SectorIdentifierURI != "" {
		if u, err := url.Parse(*client.SectorIdentifierURI); err == nil {
			return u.Host
		}
	}

	// redirect_uri のホスト部をフォールバックとして使用
	if len(client.RedirectURIs) > 0 {
		if u, err := url.Parse(client.RedirectURIs[0].URI); err == nil {
			return u.Host
		}
	}

	return ""
}

// ResolveSubject はクライアントの subject_type に基づいて sub クレームを決定する。
// public の場合はユーザー ID をそのまま返し、pairwise の場合はハッシュ化した値を返す。
func ResolveSubject(client *model.Client, userID string) string {
	if client.SubjectType == "pairwise" && client.PairwiseSalt != nil {
		sector := GetSectorIdentifier(client)
		return ComputePairwiseSub(sector, userID, *client.PairwiseSalt)
	}
	return userID
}
