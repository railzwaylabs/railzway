package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// GenerateKeyPair creates a new Ed25519 key pair.
// Returns base64 encoded private and public keys.
func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub), nil
}

// SignLicense creates a signed license JSON using the private key.
// privateKeyStr must be base64 encoded Ed25519 private key.
func SignLicense(privateKeyStr string, payload Payload) ([]byte, error) {
	privBytes, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	priv := ed25519.PrivateKey(privBytes)

	// Marshal payload to sign
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Sign
	sig := ed25519.Sign(priv, payloadBytes)

	// Create complete license
	l := License{
		Payload:   payload,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}

	return json.MarshalIndent(l, "", "  ") // Pretty print for "human readable"
}
