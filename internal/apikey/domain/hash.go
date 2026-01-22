package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashAPIKey hashes the raw API key using the same strategy as key creation.
func HashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
