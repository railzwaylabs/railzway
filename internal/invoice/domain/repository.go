package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateInvoice(ctx context.Context, inv Invoice) error
	FindInvoiceByID(ctx context.Context, orgID, invoiceID uuid.UUID) (*Invoice, error)
	FindInvoiceByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Invoice, error)
	FindInvoiceBySubscriptionPeriod(ctx context.Context, orgID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) (*Invoice, error)
	UpdateInvoice(ctx context.Context, orgID, invoiceID uuid.UUID, updates map[string]interface{}) error
	ListInvoices(ctx context.Context, orgID uuid.UUID, filter InvoiceListFilter, limit int, cursor *ListCursor) ([]*Invoice, error)
	ListInvoiceItemsByInvoice(ctx context.Context, orgID, invoiceID uuid.UUID) ([]*InvoiceItem, error)
	CreateInvoiceItems(ctx context.Context, items []InvoiceItem) error
	DeleteInvoiceItemsByInvoice(ctx context.Context, orgID, invoiceID uuid.UUID) error
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type InvoiceListFilter struct {
	CustomerID      uuid.UUID
	SubscriptionID  uuid.UUID
	Status          string
	Number          string
	PeriodStartFrom *time.Time
	PeriodStartTo   *time.Time
	IssuedFrom      *time.Time
	IssuedTo        *time.Time
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}

type UsageCharge struct {
	PlanPriceID uuid.UUID
	MeterID     uuid.UUID
	Currency    string
	Quantity    float64
	AmountCents int64
	PeriodStart time.Time
	PeriodEnd   time.Time
}
