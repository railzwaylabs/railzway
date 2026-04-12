package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/feature/domain"
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

func (r *repository) Create(ctx context.Context, feature domain.Feature) error {
	return r.db.WithContext(ctx).Create(&feature).Error
}

func (r *repository) Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Feature{}).
		Where("id = ? AND org_id = ?", id, orgID).
		Updates(updates).Error
}

func (r *repository) FindByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Feature, error) {
	var feature domain.Feature
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &feature, nil
}

func (r *repository) FindByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Feature, error) {
	var feature domain.Feature
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &feature, nil
}

func (r *repository) List(ctx context.Context, orgID uuid.UUID, filter domain.ListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Feature, error) {
	var features []*domain.Feature
	stmt := r.db.WithContext(ctx).Model(&domain.Feature{}).Where("org_id = ?", orgID)

	if filter.Code != "" {
		stmt = stmt.Where("code ILIKE ?", "%"+filter.Code+"%")
	}
	if filter.Name != "" {
		stmt = stmt.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.FeatureType != "" {
		stmt = stmt.Where("feature_type = ?", filter.FeatureType)
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

	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&features).Error; err != nil {
		return nil, err
	}
	return features, nil
}

func (r *repository) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]*domain.Feature, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var features []*domain.Feature
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id IN ?", orgID, ids).
		Find(&features).Error
	if err != nil {
		return nil, err
	}
	return features, nil
}
