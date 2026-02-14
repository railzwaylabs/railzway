package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrInvalidKey     = errors.New("vault: invalid encryption key")
	ErrInvalidPayload = errors.New("vault: invalid encrypted payload")
	ErrEncryption     = errors.New("vault: encryption failed")
	ErrDecryption     = errors.New("vault: decryption failed")
	ErrUnknownProvider = errors.New("vault: unknown provider")
	ErrMissingConfig   = errors.New("vault: missing configuration")
)

// Provider defines the interface for encryption/decryption backends.
type Provider interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

// Config holds configuration for the Vault factory.
type Config struct {
	Provider string // "aes" or "hashicorp"
	AESKey   string // 32-byte hex or raw string
	VaultAddr string
	VaultToken string
}

// NewFactory creates a Vault Provider based on configuration.
func NewFactory(cfg Config) (Provider, error) {
	switch strings.ToLower(cfg.Provider) {
	case "hashicorp":
		return newHashiCorpVault(cfg)
	case "aes", "":
		return newAESVault(cfg.AESKey)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, cfg.Provider)
	}
}

// AESVault implements Provider using AES-256-GCM.
type AESVault struct {
	key []byte
}

func newAESVault(keyStr string) (*AESVault, error) {
	if strings.TrimSpace(keyStr) == "" {
		return nil, ErrInvalidKey
	}

	// We hash the input string to ensure a 32-byte (256-bit) key for AES-256.
	// This makes it user-friendly as any string (hex, base64, or plain text)
	// can be used as the ENCRYPTION_KEY in .env.
	sum := sha256.Sum256([]byte(keyStr))
	return &AESVault{key: sum[:]}, nil
}

type EncryptedData struct {
	Version    int    `json:"v"`
	Nonce      string `json:"n"`
	Ciphertext string `json:"c"`
}

func (v *AESVault) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	payload := EncryptedData{
		Version:    1,
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	}

	return json.Marshal(payload)
}

func (v *AESVault) Decrypt(data []byte) ([]byte, error) {
	var payload EncryptedData
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, ErrInvalidPayload
	}

	if payload.Version != 1 {
		return nil, ErrInvalidPayload
	}

	nonce, err := base64.RawStdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, ErrInvalidPayload
	}

	ciphertext, err := base64.RawStdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, ErrInvalidPayload
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryption
	}

	return plaintext, nil
}

// HashiCorpVault (Stub)
type HashiCorpVault struct {
	client interface{} // Placeholder for *api.Client
}

func newHashiCorpVault(cfg Config) (*HashiCorpVault, error) {
	if cfg.VaultAddr == "" || cfg.VaultToken == "" {
		return nil, fmt.Errorf("%w: VAULT_ADDR and VAULT_TOKEN required", ErrMissingConfig)
	}
	// TODO: Initialize real HashiCorp client
	return &HashiCorpVault{}, nil
}

func (v *HashiCorpVault) Encrypt(plaintext []byte) ([]byte, error) {
	return nil, errors.New("hashicorp vault not implemented yet")
}

func (v *HashiCorpVault) Decrypt(data []byte) ([]byte, error) {
	return nil, errors.New("hashicorp vault not implemented yet")
}
