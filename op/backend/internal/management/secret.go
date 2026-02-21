package management

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateClientID generates a random client_id string (32 hex chars = 16 bytes).
func generateClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client_id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// generateClientSecret generates a random client_secret string (64 hex chars = 32 bytes).
func generateClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client_secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
