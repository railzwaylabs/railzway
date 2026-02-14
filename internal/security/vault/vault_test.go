package vault

import (
	"crypto/rand"
	"reflect"
	"testing"
)

func TestVault_AES(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	cfg := Config{
		Provider: "aes",
		AESKey:   string(key),
	}

	v, err := NewFactory(cfg)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	originalText := []byte("secret-api-key-12345")

	// Test Encryption
	encrypted, err := v.Encrypt(originalText)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Test Decryption
	decrypted, err := v.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !reflect.DeepEqual(originalText, decrypted) {
		t.Errorf("Decrypted text does not match original. Got %s, want %s", decrypted, originalText)
	}
}

func TestVault_HashiCorp_Validation(t *testing.T) {
	cfg := Config{
		Provider: "hashicorp",
		// Missing Addr/Token
	}
	_, err := NewFactory(cfg)
	if err == nil {
		t.Error("Expected error for missing HashiCorp config")
	}

	cfg.VaultAddr = "http://localhost:8200"
	cfg.VaultToken = "token"
	v, err := NewFactory(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	_, err = v.Encrypt([]byte("test"))
	if err == nil || err.Error() != "hashicorp vault not implemented yet" {
		t.Error("Expected not implemented error")
	}
}
