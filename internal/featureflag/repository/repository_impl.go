package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/featureflag/domain"
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

func (r *repository) FindFlag(ctx context.Context, orgID *uuid.UUID, key string) (*domain.FeatureFlag, error) {
	var flag domain.FeatureFlag
	stmt := r.db.WithContext(ctx).Where("key = ?", key)
	if orgID == nil {
		stmt = stmt.Where("org_id IS NULL")
	} else {
		stmt = stmt.Where("org_id = ?", *orgID)
	}
	err := stmt.First(&flag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &flag, nil
}

func (r *repository) ListFlags(ctx context.Context, orgID *uuid.UUID) ([]domain.FeatureFlag, error) {
	var flags []domain.FeatureFlag
	stmt := r.db.WithContext(ctx).Model(&domain.FeatureFlag{})
	if orgID == nil {
		stmt = stmt.Where("org_id IS NULL")
	} else {
		stmt = stmt.Where("org_id = ?", *orgID)
	}
	if err := stmt.Order("key asc").Find(&flags).Error; err != nil {
		return nil, err
	}
	return flags, nil
}

func (r *repository) UpsertFlag(ctx context.Context, flag domain.FeatureFlag) error {
	if flag.OrgID == nil {
		return r.db.WithContext(ctx).Exec(
			`INSERT INTO feature_flags (id, org_id, key, enabled, rollout, created_at, updated_at)
			 VALUES (?, NULL, ?, ?, ?, ?, ?)
			 ON CONFLICT (key) WHERE org_id IS NULL
			 DO UPDATE SET enabled = EXCLUDED.enabled,
			               rollout = EXCLUDED.rollout,
			               updated_at = EXCLUDED.updated_at`,
			flag.ID,
			flag.Key,
			flag.Enabled,
			flag.Rollout,
			flag.CreatedAt,
			flag.UpdatedAt,
		).Error
	}
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO feature_flags (id, org_id, key, enabled, rollout, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (org_id, key)
		 DO UPDATE SET enabled = EXCLUDED.enabled,
		               rollout = EXCLUDED.rollout,
		               updated_at = EXCLUDED.updated_at`,
		flag.ID,
		flag.OrgID,
		flag.Key,
		flag.Enabled,
		flag.Rollout,
		flag.CreatedAt,
		flag.UpdatedAt,
	).Error
}

func (r *repository) CreateAudit(ctx context.Context, audit domain.FeatureFlagAudit) error {
	return r.db.WithContext(ctx).Create(&audit).Error
}
