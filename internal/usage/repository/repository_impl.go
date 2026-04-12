package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/usage/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) domain.Repository {
	return &repository{db: tx}
}

func (r *repository) CreateMeter(ctx context.Context, meter domain.Meter) error {
	return r.db.WithContext(ctx).Create(&meter).Error
}

func (r *repository) UpdateMeter(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Meter{}).
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(updates).Error
}

func (r *repository) FindMeterByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Meter, error) {
	var meter domain.Meter
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&meter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &meter, nil
}

func (r *repository) FindMeterByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.Meter, error) {
	var meter domain.Meter
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&meter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &meter, nil
}

func (r *repository) FindMeterByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Meter, error) {
	var meter domain.Meter
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&meter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &meter, nil
}

func (r *repository) ListMeters(ctx context.Context, orgID uuid.UUID, filter domain.MeterListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Meter, error) {
	var meters []*domain.Meter
	stmt := r.db.WithContext(ctx).Model(&domain.Meter{}).Where("org_id = ?", orgID)
	if filter.Code != "" {
		stmt = stmt.Where("code = ?", filter.Code)
	}
	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&meters).Error
	if err != nil {
		return nil, err
	}
	return meters, nil
}

func (r *repository) CreateUsageEvent(ctx context.Context, event domain.UsageEvent) error {
	return r.db.WithContext(ctx).Create(&event).Error
}

func (r *repository) FindUsageByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.UsageEvent, error) {
	var event domain.UsageEvent
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (r *repository) ListUsageEvents(ctx context.Context, orgID uuid.UUID, filter domain.UsageListFilter, limit int, cursor *domain.ListCursor) ([]*domain.UsageEvent, error) {
	var events []*domain.UsageEvent
	stmt := r.db.WithContext(ctx).Model(&domain.UsageEvent{}).Where("org_id = ?", orgID)
	if filter.MeterID != uuid.Nil {
		stmt = stmt.Where("meter_id = ?", filter.MeterID)
	}
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.Status != "" {
		stmt = stmt.Where("status = ?", filter.Status)
	}
	if filter.RecordedFrom != nil {
		stmt = stmt.Where("recorded_at >= ?", *filter.RecordedFrom)
	}
	if filter.RecordedTo != nil {
		stmt = stmt.Where("recorded_at <= ?", *filter.RecordedTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *repository) UpdateUsageStatus(ctx context.Context, orgID, usageEventID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&domain.UsageEvent{}).
		Where("org_id = ? AND id = ?", orgID, usageEventID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
}
