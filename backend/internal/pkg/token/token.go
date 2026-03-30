package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Generate creates a cryptographically secure random token of the given byte length.
// Returns a base64url-encoded string (no padding).
// Default 32 bytes = 256 bits of entropy.
func Generate(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 32
	}
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
