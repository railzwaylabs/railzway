package domain

import (
	"context"
	"net/http"

	"github.com/bwmarrin/snowflake"
)

type PaymentAdapter interface {
	// Webhook handling
	Verify(ctx context.Context, payload []byte, headers http.Header) error
	Parse(ctx context.Context, payload []byte) (*PaymentEvent, error)
	
	// Payment method management
	AttachPaymentMethod(ctx context.Context, customerProviderID, token string) (*PaymentMethodDetails, error)
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethodDetails, error)
	ListPaymentMethods(ctx context.Context, customerProviderID string) ([]*PaymentMethodDetails, error)
}

type AdapterConfig struct {
	OrgID    snowflake.ID
	Provider string
	Config   map[string]any
}

type AdapterFactory interface {
	Provider() string
	NewAdapter(config AdapterConfig) (PaymentAdapter, error)
}
