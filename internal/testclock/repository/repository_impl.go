package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/testclock/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *repository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*domain.TestClock, error) {
	var clock domain.TestClock
	err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("created_at DESC, id DESC").
		First(&clock).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &clock, nil
}

func (r *repository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*domain.TestClock, error) {
	var clock domain.TestClock
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&clock).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &clock, nil
}

func (r *repository) Create(ctx context.Context, clock domain.TestClock) error {
	return r.db.WithContext(ctx).Create(&clock).Error
}

func (r *repository) Upsert(ctx context.Context, clock domain.TestClock) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "status", "clock_time", "updated_at"}),
	}).Create(&clock).Error
}

func (r *repository) Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.TestClock{}).
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(updates).Error
}

func (r *repository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]domain.TestClock, error) {
	var clocks []domain.TestClock
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("created_at DESC, id DESC").
		Find(&clocks).Error; err != nil {
		return nil, err
	}
	return clocks, nil
}

func (r *repository) ListActive(ctx context.Context) ([]domain.TestClock, error) {
	var clocks []domain.TestClock
	if err := r.db.WithContext(ctx).
		Where("status = ?", domain.StatusActive).
		Find(&clocks).Error; err != nil {
		return nil, err
	}
	return clocks, nil
}
