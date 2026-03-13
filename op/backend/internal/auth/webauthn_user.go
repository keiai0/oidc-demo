package auth

import (
	"encoding/base64"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// webauthnUser は go-webauthn/webauthn の webauthn.User インターフェースを満たすアダプター。
type webauthnUser struct {
	id          uuid.UUID
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id[:] }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// toWebAuthnCredentials は model.WebAuthnCredential のスライスを webauthn.Credential のスライスに変換する。
func toWebAuthnCredentials(creds []model.WebAuthnCredential) []webauthn.Credential {
	result := make([]webauthn.Credential, 0, len(creds))
	for _, c := range creds {
		credID, err := base64.RawURLEncoding.DecodeString(c.CredentialID)
		if err != nil {
			continue
		}
		pubKey, err := base64.StdEncoding.DecodeString(c.PublicKey)
		if err != nil {
			continue
		}
		aaguid, _ := base64.StdEncoding.DecodeString(c.AAGUID)

		result = append(result, webauthn.Credential{
			ID:              credID,
			PublicKey:       pubKey,
			AttestationType: c.AttestationType,
			Flags: webauthn.CredentialFlags{
				BackupEligible: c.BackupEligible,
				BackupState:    c.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    aaguid,
				SignCount: c.SignCount,
			},
			Transport: []protocol.AuthenticatorTransport{},
		})
	}
	return result
}
