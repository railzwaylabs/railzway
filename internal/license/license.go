package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid license signature")
	ErrExpired          = errors.New("license expired")
	ErrInvalidFormat    = errors.New("invalid license format")
)

// License represents the signed license structure.
type License struct {
	Payload   Payload `json:"payload"`
	Signature string  `json:"signature"`
}

// Payload contains the actual license claims.
type Payload struct {
	OrgID     string    `json:"org_id"`
	Features  []string  `json:"features"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	IssuedBy  string    `json:"issued_by,omitempty"`
	MaxSeats  int       `json:"max_seats,omitempty"`
}

// Verifier handles license verification.
type Verifier struct {
	publicKey ed25519.PublicKey
}

// NewVerifier creates a new Verifier with the given Ed25519 public key.
// The publicKey must be a 32-byte Ed25519 public key.
func NewVerifier(publicKey []byte) *Verifier {
	return &Verifier{
		publicKey: publicKey,
	}
}

// ParseAndVerify parses a JSON license, validates the signature against the public key,
// and checks for expiration.
func (v *Verifier) ParseAndVerify(licenseJSON []byte) (*Payload, error) {
	var l License
	if err := json.Unmarshal(licenseJSON, &l); err != nil {
		return nil, ErrInvalidFormat
	}

	// 1. Decode Signature
	sig, err := base64.StdEncoding.DecodeString(l.Signature)
	if err != nil {
		return nil, ErrInvalidFormat
	}

	// 2. Re-encode payload to bytes for verification to ensure consistency
	// In a real-world scenario, you might want to sign the raw original bytes of the payload
	// part, but since we receive full JSON, we can assume the structure.
	// However, JSON canonicalization is tricky. A better approach for the signature
	// is typically signing the base64url encoded payload (like JWT).
	//
	// Given the strategy doc example used a wrapped structure:
	// { "payload": {...}, "signature": "..." }
	// We need to be careful about what exactly was signed.
	//
	// Strategy: The strategy says "signed, human-readable license file".
	// The robust way is to sign the canonical JSON or a byte representation.
	// For simplicity and typical use, we'll serialize the Payload struct back to JSON
	// to verify. NOTE: This depends on stable JSON marshaling (e.g. sorted keys).
	// Go's json.Marshal sorts map keys, which helps.
	payloadBytes, err := json.Marshal(l.Payload)
	if err != nil {
		return nil, err
	}

	if !ed25519.Verify(v.publicKey, payloadBytes, sig) {
		return nil, ErrInvalidSignature
	}

	// 3. Check Expiration
	if time.Now().After(l.Payload.ExpiresAt) {
		return nil, ErrExpired
	}

	return &l.Payload, nil
}
