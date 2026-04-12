package publiclink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid_token")
	ErrExpiredToken = errors.New("expired_token")
	ErrMissingKey   = errors.New("missing_link_secret")
)

func BuildInvoiceToken(invoiceID, orgID uuid.UUID, secret string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrMissingKey
	}
	if invoiceID == uuid.Nil || orgID == uuid.Nil {
		return "", ErrInvalidToken
	}
	expiresAt := time.Now().UTC().Add(ttl).Unix()
	payload := fmt.Sprintf("%s:%s:%d", invoiceID.String(), orgID.String(), expiresAt)
	sig := sign(payload, secret)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + sig)), nil
}

func ParseInvoiceToken(token, secret string, now time.Time) (uuid.UUID, uuid.UUID, time.Time, error) {
	if strings.TrimSpace(secret) == "" {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrMissingKey
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 4 {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}
	payload := strings.Join(parts[:3], ":")
	sig := parts[3]
	if !verify(payload, sig, secret) {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}

	invoiceID, err := uuid.Parse(parts[0])
	if err != nil || invoiceID == uuid.Nil {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}
	orgID, err := uuid.Parse(parts[1])
	if err != nil || orgID == uuid.Nil {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}
	expUnix, err := parseInt64(parts[2])
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, ErrInvalidToken
	}
	exp := time.Unix(expUnix, 0).UTC()
	if now.UTC().After(exp) {
		return uuid.Nil, uuid.Nil, exp, ErrExpiredToken
	}
	return invoiceID, orgID, exp, nil
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verify(payload, sig, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	decoded, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, decoded)
}

func parseInt64(value string) (int64, error) {
	var out int64
	_, err := fmt.Sscanf(value, "%d", &out)
	if err != nil {
		return 0, err
	}
	return out, nil
}
