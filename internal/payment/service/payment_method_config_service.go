package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	providerservice "github.com/railzwaylabs/railzway/internal/providers/payment/domain"
	"gorm.io/gorm"
)

type PaymentMethodConfigServiceImpl struct {
	db              *gorm.DB
	providerService providerservice.Service
}

func NewPaymentMethodConfigService(db *gorm.DB, providerService providerservice.Service) domain.PaymentMethodConfigService {
	return &PaymentMethodConfigServiceImpl{
		db:              db,
		providerService: providerService,
	}
}

func (s *PaymentMethodConfigServiceImpl) GetAvailablePaymentMethods(
	ctx context.Context,
	orgID snowflake.ID,
	country, currency string,
) ([]*domain.PaymentMethodConfig, error) {
	var configs []*domain.PaymentMethodConfig

	// Get all active configs for org
	if err := s.db.Where("org_id = ? AND is_active = true", orgID).
		Order("priority DESC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	// Filter by availability rules
	available := []*domain.PaymentMethodConfig{}
	for _, config := range configs {
		if s.isAvailable(config, country, currency) {
			// Populate transient PublicKey from provider config
			if s.providerService != nil {
				// We need to fetch the provider config (which is cached or fast hopefully)
				// Note: Ideally this is batched or cached, but for now N+1 is acceptable for low N configs
				providerCfg, err := s.providerService.GetActiveProviderConfig(ctx, orgID, config.Provider)
				if err == nil && providerCfg != nil {
					var cfgMap map[string]any
					if err := json.Unmarshal(providerCfg.Config, &cfgMap); err == nil {
						if pk, ok := cfgMap["public_key"].(string); ok {
							config.PublicKey = pk
						} else if pk, ok := cfgMap["publishable_key"].(string); ok {
							config.PublicKey = pk
						}
					}
				}
			}
			available = append(available, config)
		}
	}

	return available, nil
}

func (s *PaymentMethodConfigServiceImpl) isAvailable(config *domain.PaymentMethodConfig, country, currency string) bool {
	var rules domain.AvailabilityRules
	if err := json.Unmarshal(config.AvailabilityRules, &rules); err != nil {
		return false
	}

	// Check country
	if !contains(rules.Countries, "*") && !contains(rules.Countries, country) {
		return false
	}

	// Check currency
	if len(rules.Currencies) > 0 && !contains(rules.Currencies, currency) {
		return false
	}

	return true
}

func (s *PaymentMethodConfigServiceImpl) GetPaymentMethodConfig(
	ctx context.Context,
	orgID snowflake.ID,
	methodName string,
) (*domain.PaymentMethodConfig, error) {
	var config domain.PaymentMethodConfig
	if err := s.db.Where("org_id = ? AND method_name = ? AND is_active = true", orgID, methodName).
		First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *PaymentMethodConfigServiceImpl) ListPaymentMethodConfigs(
	ctx context.Context,
	orgID snowflake.ID,
) ([]*domain.PaymentMethodConfig, error) {
	var configs []*domain.PaymentMethodConfig
	if err := s.db.Where("org_id = ?", orgID).
		Order("priority DESC, method_name ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *PaymentMethodConfigServiceImpl) UpsertPaymentMethodConfig(
	ctx context.Context,
	config *domain.PaymentMethodConfig,
) error {
	config.UpdatedAt = time.Now().UTC()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now().UTC()
	}

	return s.db.Save(config).Error
}

func (s *PaymentMethodConfigServiceImpl) DeletePaymentMethodConfig(
	ctx context.Context,
	orgID, configID snowflake.ID,
) error {
	return s.db.Where("org_id = ? AND id = ?", orgID, configID).
		Delete(&domain.PaymentMethodConfig{}).Error
}

func (s *PaymentMethodConfigServiceImpl) TogglePaymentMethodConfig(
	ctx context.Context,
	orgID, configID snowflake.ID,
	isActive bool,
) error {
	return s.db.Model(&domain.PaymentMethodConfig{}).
		Where("org_id = ? AND id = ?", orgID, configID).
		Update("is_active", isActive).Error
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
