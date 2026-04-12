package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type MeterListFilter struct {
	Code   string
	Name   string
	Active *bool
}

type UsageListFilter struct {
	MeterID      uuid.UUID
	CustomerID   uuid.UUID
	Status       string
	RecordedFrom *time.Time
	RecordedTo   *time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateMeter(ctx context.Context, meter Meter) error
	UpdateMeter(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindMeterByID(ctx context.Context, orgID, id uuid.UUID) (*Meter, error)
	FindMeterByCode(ctx context.Context, orgID uuid.UUID, code string) (*Meter, error)
	FindMeterByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Meter, error)
	ListMeters(ctx context.Context, orgID uuid.UUID, filter MeterListFilter, limit int, cursor *ListCursor) ([]*Meter, error)

	CreateUsageEvent(ctx context.Context, event UsageEvent) error
	FindUsageByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*UsageEvent, error)
	ListUsageEvents(ctx context.Context, orgID uuid.UUID, filter UsageListFilter, limit int, cursor *ListCursor) ([]*UsageEvent, error)
	UpdateUsageStatus(ctx context.Context, orgID, usageEventID uuid.UUID, status string) error
}
