package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/payment/adapters"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	providerservice "github.com/railzwaylabs/railzway/internal/providers/payment/domain"
	"gorm.io/gorm"
)

type PaymentMethodServiceImpl struct {
	db              *gorm.DB
	idGen           *snowflake.Node
	adapterRegistry *adapters.Registry
	providerService providerservice.Service
}

func NewPaymentMethodService(
	db *gorm.DB,
	idGen *snowflake.Node,
	adapterRegistry *adapters.Registry,
	providerService providerservice.Service,
) domain.PaymentMethodService {
	return &PaymentMethodServiceImpl{
		db:              db,
		idGen:           idGen,
		adapterRegistry: adapterRegistry,
		providerService: providerService,
	}
}

func (s *PaymentMethodServiceImpl) AttachPaymentMethod(
	ctx context.Context,
	customerID snowflake.ID,
	provider, token string,
) (*domain.PaymentMethod, error) {
	// 1. Get customer and their provider customer ID
	var customer struct {
		OrgID              snowflake.ID
		ProviderCustomerID string
	}
	if err := s.db.Table("customers").
		Select("org_id, provider_customer_id").
		Where("id = ?", customerID).
		First(&customer).Error; err != nil {
		return nil, err
	}

	// 2. Get provider config and create adapter
	providerConfig, err := s.providerService.GetActiveProviderConfig(ctx, customer.OrgID, provider)
	if err != nil {
		return nil, err
	}

	var configMap map[string]any
	if err := json.Unmarshal(providerConfig.Config, &configMap); err != nil {
		return nil, err
	}

	adapter, err := s.adapterRegistry.NewAdapter(provider, domain.AdapterConfig{
		OrgID:    customer.OrgID,
		Provider: provider,
		Config:   configMap,
	})
	if err != nil {
		return nil, err
	}

	// 3. Attach payment method via provider adapter
	pmDetails, err := adapter.AttachPaymentMethod(ctx, customer.ProviderCustomerID, token)
	if err != nil {
		return nil, err
	}

	// 4. Save to database
	pm := &domain.PaymentMethod{
		ID:                      s.idGen.Generate(),
		CustomerID:              customerID,
		Provider:                provider,
		ProviderPaymentMethodID: pmDetails.ID,
		Type:                    pmDetails.Type,
		Last4:                   pmDetails.Last4,
		Brand:                   pmDetails.Brand,
		ExpMonth:                pmDetails.ExpMonth,
		ExpYear:                 pmDetails.ExpYear,
		IsDefault:               false,
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
	}

	if err := s.db.Create(pm).Error; err != nil {
		return nil, err
	}

	// 5. If this is the first payment method, make it default
	var count int64
	s.db.Model(&domain.PaymentMethod{}).Where("customer_id = ?", customerID).Count(&count)
	if count == 1 {
		pm.IsDefault = true
		s.db.Save(pm)
	}

	return pm, nil
}

func (s *PaymentMethodServiceImpl) ListPaymentMethods(
	ctx context.Context,
	customerID snowflake.ID,
) ([]*domain.PaymentMethod, error) {
	var pms []*domain.PaymentMethod
	if err := s.db.Where("customer_id = ?", customerID).
		Order("is_default DESC, created_at DESC").
		Find(&pms).Error; err != nil {
		return nil, err
	}
	return pms, nil
}

func (s *PaymentMethodServiceImpl) SetDefaultPaymentMethod(
	ctx context.Context,
	customerID, paymentMethodID snowflake.ID,
) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Verify ownership
		var pm domain.PaymentMethod
		if err := tx.Where("id = ? AND customer_id = ?", paymentMethodID, customerID).
			First(&pm).Error; err != nil {
			return err
		}

		// Unset current default
		tx.Model(&domain.PaymentMethod{}).
			Where("customer_id = ? AND is_default = true", customerID).
			Update("is_default", false)

		// Set new default
		pm.IsDefault = true
		pm.UpdatedAt = time.Now().UTC()
		return tx.Save(&pm).Error
	})
}

func (s *PaymentMethodServiceImpl) DetachPaymentMethod(
	ctx context.Context,
	customerID, paymentMethodID snowflake.ID,
) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var pm domain.PaymentMethod
		if err := tx.Where("id = ? AND customer_id = ?", paymentMethodID, customerID).
			First(&pm).Error; err != nil {
			return err
		}

		// Get provider config
		var customer struct {
			OrgID snowflake.ID
		}
		if err := tx.Table("customers").
			Select("org_id").
			Where("id = ?", customerID).
			First(&customer).Error; err != nil {
			return err
		}

		providerConfig, err := s.providerService.GetActiveProviderConfig(ctx, customer.OrgID, pm.Provider)
		if err != nil {
			return err
		}

		var configMap map[string]any
		if err := json.Unmarshal(providerConfig.Config, &configMap); err != nil {
			return err
		}

		adapter, err := s.adapterRegistry.NewAdapter(pm.Provider, domain.AdapterConfig{
			OrgID:    customer.OrgID,
			Provider: pm.Provider,
			Config:   configMap,
		})
		if err != nil {
			return err
		}

		// Detach from provider
		if err := adapter.DetachPaymentMethod(ctx, pm.ProviderPaymentMethodID); err != nil {
			// Log error but continue with deletion
		}

		// Delete from database
		return tx.Delete(&pm).Error
	})
}

func (s *PaymentMethodServiceImpl) GetDefaultPaymentMethod(
	ctx context.Context,
	customerID snowflake.ID,
) (*domain.PaymentMethod, error) {
	var pm domain.PaymentMethod
	if err := s.db.Where("customer_id = ? AND is_default = true", customerID).
		First(&pm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pm, nil
}
