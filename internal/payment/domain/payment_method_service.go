package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
)

// PaymentMethodService manages customer payment methods
type PaymentMethodService interface {
	// AttachPaymentMethod attaches a tokenized payment method to a customer
	AttachPaymentMethod(ctx context.Context, customerID snowflake.ID, provider, token string) (*PaymentMethod, error)
	
	// ListPaymentMethods lists all payment methods for a customer
	ListPaymentMethods(ctx context.Context, customerID snowflake.ID) ([]*PaymentMethod, error)
	
	// SetDefaultPaymentMethod sets a payment method as the default
	SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID snowflake.ID) error
	
	// DetachPaymentMethod removes a payment method
	DetachPaymentMethod(ctx context.Context, customerID, paymentMethodID snowflake.ID) error
	
	// GetDefaultPaymentMethod gets the default payment method for a customer
	GetDefaultPaymentMethod(ctx context.Context, customerID snowflake.ID) (*PaymentMethod, error)
}

// PaymentMethodConfigService manages payment method configurations
type PaymentMethodConfigService interface {
	// GetAvailablePaymentMethods returns available payment methods based on country and currency
	GetAvailablePaymentMethods(ctx context.Context, orgID snowflake.ID, country, currency string) ([]*PaymentMethodConfig, error)
	
	// GetPaymentMethodConfig gets a specific payment method configuration
	GetPaymentMethodConfig(ctx context.Context, orgID snowflake.ID, methodName string) (*PaymentMethodConfig, error)
	
	// ListPaymentMethodConfigs lists all payment method configurations for an org
	ListPaymentMethodConfigs(ctx context.Context, orgID snowflake.ID) ([]*PaymentMethodConfig, error)
	
	// UpsertPaymentMethodConfig creates or updates a payment method configuration
	UpsertPaymentMethodConfig(ctx context.Context, config *PaymentMethodConfig) error
	
	// DeletePaymentMethodConfig deletes a payment method configuration
	DeletePaymentMethodConfig(ctx context.Context, orgID, configID snowflake.ID) error
	
	// TogglePaymentMethodConfig enables or disables a payment method configuration
	TogglePaymentMethodConfig(ctx context.Context, orgID, configID snowflake.ID, isActive bool) error
}
