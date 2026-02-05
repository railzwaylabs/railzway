package domain

import (
	"context"
	"errors"
	"net/http"

	"github.com/bwmarrin/snowflake"
)

type Service interface {
	IngestWebhook(ctx context.Context, provider string, payload []byte, headers http.Header) error
}

type CheckoutService interface {
	CreateSession(ctx context.Context, input CheckoutSessionInput) (*CheckoutSession, error)
	GetSession(ctx context.Context, id snowflake.ID) (*CheckoutSession, error)
	GetLineItems(ctx context.Context, sessionID snowflake.ID) ([]LineItem, error)
	ExpireSession(ctx context.Context, id snowflake.ID) (*CheckoutSession, error)
	CompleteSession(ctx context.Context, provider, providerSessionID string) (*CheckoutSession, error)
	VerifyAndComplete(ctx context.Context, sessionID string) (*CheckoutSession, error)
}

var (
	ErrInvalidProvider       = errors.New("invalid_provider")
	ErrProviderNotFound      = errors.New("provider_not_found")
	ErrInvalidSignature      = errors.New("invalid_signature")
	ErrInvalidPayload        = errors.New("invalid_payload")
	ErrInvalidEvent          = errors.New("invalid_event")
	ErrInvalidOrganization   = errors.New("invalid_organization")
	ErrEventIgnored          = errors.New("event_ignored")
	ErrInvalidCustomer       = errors.New("invalid_customer")
	ErrInvalidAmount         = errors.New("invalid_amount")
	ErrInvalidCurrency       = errors.New("invalid_currency")
	ErrInvalidConfig         = errors.New("invalid_config")
	ErrEventAlreadyProcessed = errors.New("event_already_processed")
	ErrPaymentMethodNotFound = errors.New("payment_method_not_found")
	ErrInvalidPaymentMethod  = errors.New("invalid_payment_method")
)
