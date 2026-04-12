package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	ListPayments(ctx context.Context, orgID uuid.UUID, filter PaymentListFilter, limit int, cursor *ListCursor) ([]*Payment, error)
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type PaymentListFilter struct {
	CustomerID  uuid.UUID
	InvoiceID   uuid.UUID
	Status      string
	Provider    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}
