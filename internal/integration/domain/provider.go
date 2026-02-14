package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
)

// NotificationProvider defines the interface for notification channels (Slack, Email, etc.)
type NotificationProvider interface {
	Send(ctx context.Context, input NotificationInput) error
}

type NotificationInput struct {
	ConnectionID snowflake.ID
	ChannelID    string // e.g., Slack Channel ID, Email Address
	TemplateID   string // e.g., "invoice.paid"
	Data         map[string]any
}

// AccountingProvider defines the interface for accounting/ERP systems (Xero, NetSuite)
type AccountingProvider interface {
	SyncInvoice(ctx context.Context, input SyncInvoiceInput) error
	SyncPayment(ctx context.Context, input SyncPaymentInput) error
}

type SyncInvoiceInput struct {
	ConnectionID snowflake.ID
	InvoiceID    snowflake.ID
	// We might pass the full invoice object here or fetch it inside the provider
}

type SyncPaymentInput struct {
	ConnectionID snowflake.ID
	PaymentID    snowflake.ID
}
