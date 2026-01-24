package license

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseVerification(t *testing.T) {
	privStr, pubStr, err := GenerateKeyPair()
	require.NoError(t, err)

	pubBytes, err := base64.StdEncoding.DecodeString(pubStr)
	require.NoError(t, err)

	verifier := NewVerifier(pubBytes)

	t.Run("Valid License", func(t *testing.T) {
		payload := Payload{
			OrgID:     "acme-corp",
			Features:  []string{"sso", "audit_log"},
			ExpiresAt: time.Now().Add(24 * time.Hour),
			IssuedAt:  time.Now(),
		}

		licenseJSON, err := SignLicense(privStr, payload)
		require.NoError(t, err)

		parsed, err := verifier.ParseAndVerify(licenseJSON)
		require.NoError(t, err)
		assert.Equal(t, "acme-corp", parsed.OrgID)
		assert.Contains(t, parsed.Features, "sso")
	})

	t.Run("Expired License", func(t *testing.T) {
		payload := Payload{
			OrgID:     "expired-corp",
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired yesterday
		}

		licenseJSON, err := SignLicense(privStr, payload)
		require.NoError(t, err)

		_, err = verifier.ParseAndVerify(licenseJSON)
		assert.ErrorIs(t, err, ErrExpired)
	})

	t.Run("Tampered License", func(t *testing.T) {
		payload := Payload{
			OrgID:     "evil-corp",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		licenseJSON, err := SignLicense(privStr, payload)
		require.NoError(t, err)

		// Tamper with the JSON
		tamperedJSON := []byte(string(licenseJSON) + " ") // Whitespace might be ignored by JSON decoder but signature over bytes? 
		// Actually, our current implementation re-marshals the payload from the decoded struct.
		// So modifying the JSON structure *values* is the attack we check.

		// Let's modify the OrgID in the JSON string manually
		// This requires simple string manipulation since we don't want to re-sign
		// We replace "evil-corp" with "good-corp" without updating signature
		// Note: The signature is base64, usually at the end.
		
		// If we modify the payload, the signature verification must fail.
		// Since we unmarshal -> marshal -> verify, changing the input JSON values changes the marshaled bytes.
		
		// Let's try to just use a different payload but same signature
		badPayload := Payload{
			OrgID: "hacked-corp", 
			ExpiresAt: time.Now().Add(time.Hour),
		}
		// Create a license struct with valid signature from previous, but bad payload
		var l License
		_ = json.Unmarshal(licenseJSON, &l)
		l.Payload = badPayload // Swapped!
		
		badLicenseJSON, _ := json.Marshal(l)
		
		_, err = verifier.ParseAndVerify(badLicenseJSON)
		assert.ErrorIs(t, err, ErrInvalidSignature)
	})
}
