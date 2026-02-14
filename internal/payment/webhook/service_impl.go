package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/payment/adapters"
	disputedomain "github.com/railzwaylabs/railzway/internal/payment/dispute/domain"
	disputeservice "github.com/railzwaylabs/railzway/internal/payment/dispute/service"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	paymentservice "github.com/railzwaylabs/railzway/internal/payment/service"
	paymentproviderdomain "github.com/railzwaylabs/railzway/internal/providers/payment/domain"
	"github.com/railzwaylabs/railzway/internal/security/vault"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB          *gorm.DB
	Log         *zap.Logger
	PaymentSvc  *paymentservice.Service
	CheckoutSvc domain.CheckoutService
	DisputeSvc  *disputeservice.Service
	Adapters    *adapters.Registry
	Cfg         config.Config
	Vault       vault.Provider
}

type Service struct {
	db          *gorm.DB
	log         *zap.Logger
	paymentSvc  *paymentservice.Service
	checkoutSvc domain.CheckoutService
	disputeSvc  *disputeservice.Service
	adapters    *adapters.Registry
	vault       vault.Provider
}

type providerConfigRow struct {
	OrgID  snowflake.ID
	Config datatypes.JSON
}

func NewService(p Params) paymentdomain.Service {
	return &Service{
		db:          p.DB,
		log:         p.Log.Named("payment.webhook"),
		paymentSvc:  p.PaymentSvc,
		checkoutSvc: p.CheckoutSvc,
		disputeSvc:  p.DisputeSvc,
		adapters:    p.Adapters,
		vault:       p.Vault,
	}
}

func (s *Service) IngestWebhook(ctx context.Context, provider string, payload []byte, headers http.Header) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return paymentdomain.ErrInvalidProvider
	}
	if s.adapters == nil || !s.adapters.ProviderExists(provider) {
		return paymentdomain.ErrProviderNotFound
	}
	if !json.Valid(payload) {
		return paymentdomain.ErrInvalidPayload
	}

	configs, err := s.listActiveConfigs(ctx, provider)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		return paymentdomain.ErrProviderNotFound
	}

	// Log incoming webhook for debugging
	s.log.Info("processing webhook",
		zap.String("provider", provider),
		zap.Int("payload_size", len(payload)),
		zap.Int("config_count", len(configs)))

	_, paymentEvent, disputeEvent, err := s.matchAdapter(ctx, provider, payload, headers, configs)
	if err != nil {
		if errors.Is(err, paymentdomain.ErrEventIgnored) {
			s.log.Debug("webhook event ignored",
				zap.String("provider", provider))
			return nil
		}
		if errors.Is(err, paymentdomain.ErrInvalidCustomer) {
			s.log.Warn("payment webhook missing customer mapping", zap.String("provider", provider))
		}
		// Log error with more context
		s.log.Error("webhook processing failed",
			zap.String("provider", provider),
			zap.Error(err),
			zap.Int("payload_size", len(payload)))
		return err
	}

	if disputeEvent != nil {
		if s.disputeSvc == nil {
			return errors.New("dispute_service_unavailable")
		}
		if disputeEvent.RawPayload == nil {
			disputeEvent.RawPayload = maskPayload(payload)
		}
		return s.disputeSvc.ProcessEvent(ctx, disputeEvent)
	}

	if paymentEvent == nil {
		return paymentdomain.ErrInvalidSignature
	}
	if s.paymentSvc == nil {
		return errors.New("payment_service_unavailable")
	}
	if paymentEvent.RawPayload == nil {
		paymentEvent.RawPayload = payload
	}

	masked := maskPayload(payload)

	// Try to complete checkout session if applicable
	// We do this speculatively for relevant event types
	if paymentEvent.Type == paymentdomain.EventTypeCheckoutSessionCompleted ||
		paymentEvent.Type == paymentdomain.EventTypePaymentSucceeded {

		if s.checkoutSvc != nil {
			// Use ProviderPaymentID as session ID (mapped in adapters)
			_, err := s.checkoutSvc.CompleteSession(ctx, provider, paymentEvent.ProviderPaymentID)
			if err != nil && !errors.Is(err, paymentdomain.ErrCheckoutSessionNotFound) {
				// Log error but don't fail the webhook processing itself,
				// as the main payment processing might still need to succeed?
				// Actually if session completion fails (e.g. DB error), we might accept it to be safe
				// or retry.
				// For now log error.
				s.log.Error("failed to complete checkout session",
					zap.String("provider", provider),
					zap.String("provider_session_id", paymentEvent.ProviderPaymentID),
					zap.Error(err))
			}
		}
	}

	return s.paymentSvc.ProcessEvent(ctx, paymentEvent, masked)
}

func (s *Service) listActiveConfigs(ctx context.Context, provider string) ([]providerConfigRow, error) {
	var rows []providerConfigRow
	err := s.db.WithContext(ctx).Raw(
		`SELECT org_id, config
		 FROM payment_provider_configs
		 WHERE provider = ? AND is_active = TRUE`,
		provider,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) matchAdapter(
	ctx context.Context,
	provider string,
	payload []byte,
	headers http.Header,
	configs []providerConfigRow,
) (paymentdomain.PaymentAdapter, *paymentdomain.PaymentEvent, *disputedomain.DisputeEvent, error) {
	var configErr error
	for _, cfg := range configs {
		decrypted, err := s.decryptConfig(cfg.Config)
		if err != nil {
			if errors.Is(err, paymentproviderdomain.ErrEncryptionKeyMissing) {
				return nil, nil, nil, err
			}
			configErr = err
			continue
		}

		adapter, err := s.adapters.NewAdapter(provider, paymentdomain.AdapterConfig{
			OrgID:    cfg.OrgID,
			Provider: provider,
			Config:   decrypted,
		})
		if err != nil {
			configErr = err
			continue
		}

		if err := adapter.Verify(ctx, payload, headers); err != nil {
			if errors.Is(err, paymentdomain.ErrInvalidSignature) {
				continue
			}
			return nil, nil, nil, err
		}

		if disputeAdapter, ok := adapter.(disputedomain.DisputeAdapter); ok {
			disputeEvent, err := disputeAdapter.ParseDispute(ctx, payload)
			if err == nil {
				disputeEvent.Provider = provider
				disputeEvent.OrgID = cfg.OrgID
				return adapter, nil, disputeEvent, nil
			}
			if !errors.Is(err, paymentdomain.ErrEventIgnored) {
				return nil, nil, nil, err
			}
		}

		paymentEvent, err := adapter.Parse(ctx, payload)
		if err != nil {
			if errors.Is(err, paymentdomain.ErrEventIgnored) {
				return adapter, nil, nil, err
			}
			return nil, nil, nil, err
		}
		paymentEvent.Provider = provider
		paymentEvent.OrgID = cfg.OrgID
		return adapter, paymentEvent, nil, nil
	}

	if configErr != nil {
		return nil, nil, nil, configErr
	}
	return nil, nil, nil, paymentdomain.ErrInvalidSignature
}

func (s *Service) decryptConfig(encrypted datatypes.JSON) (map[string]any, error) {
	if s.vault == nil {
		return nil, paymentproviderdomain.ErrEncryptionKeyMissing
	}
	if len(encrypted) == 0 {
		return nil, paymentdomain.ErrInvalidConfig
	}

	decrypted, err := s.vault.Decrypt(encrypted)
	if err != nil {
		return nil, paymentdomain.ErrInvalidConfig
	}

	var out map[string]any
	if err := json.Unmarshal(decrypted, &out); err != nil {
		return nil, paymentdomain.ErrInvalidConfig
	}
	if len(out) == 0 {
		return nil, paymentdomain.ErrInvalidConfig
	}
	return out, nil
}

func maskPayload(raw []byte) []byte {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw // fallback
	}
	maskMap(obj)
	masked, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return masked
}

func maskMap(m map[string]any) {
	for k, v := range m {
		// keys to mask
		switch strings.ToLower(k) {
		case "card", "billing_details", "shipping_details", "payment_method_details":
			m[k] = "***"
		default:
			if nested, ok := v.(map[string]any); ok {
				maskMap(nested)
			} else if arr, ok := v.([]any); ok {
				for _, item := range arr {
					if itemMap, ok := item.(map[string]any); ok {
						maskMap(itemMap)
					}
				}
			}
		}
	}
}
