package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/config"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"

	publicinvoicedomain "github.com/smallbiznis/railzway/internal/publicinvoice/domain"
	"go.uber.org/fx"
)

type TokenParams struct {
	fx.In

	Repo  publicinvoicedomain.PublicInvoiceTokenRepository
	GenID *snowflake.Node
	Cfg   config.Config
}

type TokenService struct {
	repo   publicinvoicedomain.PublicInvoiceTokenRepository
	genID  *snowflake.Node
	encKey []byte
}

func NewTokenService(p TokenParams) publicinvoicedomain.PublicInvoiceTokenService {
	secret := strings.TrimSpace(p.Cfg.PaymentProviderConfigSecret)
	var key []byte
	if secret != "" {
		sum := sha256.Sum256([]byte(secret))
		key = sum[:]
	}

	return &TokenService{
		repo:   p.Repo,
		genID:  p.GenID,
		encKey: key,
	}
}

func (s *TokenService) EnsureForInvoice(
	ctx context.Context,
	invoice invoicedomain.Invoice,
) (publicinvoicedomain.PublicInvoiceToken, error) {
	if invoice.ID == 0 || invoice.OrgID == 0 {
		return publicinvoicedomain.PublicInvoiceToken{}, publicinvoicedomain.ErrInvariantViolation
	}

	status := strings.ToUpper(strings.TrimSpace(string(invoice.Status)))
	switch status {
	case string(invoicedomain.InvoiceStatusVoid):
		return publicinvoicedomain.PublicInvoiceToken{}, publicinvoicedomain.ErrInvoiceVoided
	case string(invoicedomain.InvoiceStatusDraft):
		return publicinvoicedomain.PublicInvoiceToken{}, publicinvoicedomain.ErrInvoiceNotFinalized
	case string(invoicedomain.InvoiceStatusFinalized), "ISSUED", "PAID":
		// allowed
	default:
		return publicinvoicedomain.PublicInvoiceToken{}, publicinvoicedomain.ErrInvoiceNotFinalized
	}

	existing, err := s.repo.FindActiveByInvoiceID(ctx, invoice.ID)
	if err != nil {
		return publicinvoicedomain.PublicInvoiceToken{}, err
	}
	if existing != nil {
		if strings.TrimSpace(existing.TokenHash) == "" {
			return publicinvoicedomain.PublicInvoiceToken{}, publicinvoicedomain.ErrInvariantViolation
		}
		return *existing, nil
	}

	rawToken, err := generateToken()
	if err != nil {
		return publicinvoicedomain.PublicInvoiceToken{}, err
	}

	now := time.Now().UTC()
	newToken := publicinvoicedomain.PublicInvoiceToken{
		ID:        s.genID.Generate(),
		OrgID:     invoice.OrgID,
		InvoiceID: invoice.ID,
		CreatedAt: now,
	}

	if len(s.encKey) > 0 {
		encrypted, err := encryptToken(s.encKey, rawToken)
		if err == nil {
			newToken.TokenHash = encrypted
		}
	}

	if err := s.repo.Create(ctx, newToken); err != nil {
		fallback, fetchErr := s.repo.FindActiveByInvoiceID(ctx, invoice.ID)
		if fetchErr == nil && fallback != nil && strings.TrimSpace(fallback.TokenHash) != "" {
			return *fallback, nil
		}
		return publicinvoicedomain.PublicInvoiceToken{}, err
	}

	newToken.TokenHash = rawToken
	return newToken, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func encryptToken(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}
