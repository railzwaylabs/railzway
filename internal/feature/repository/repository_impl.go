package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/feature/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() domain.Repository {
	return &repo{}
}

func (r *repo) Create(ctx context.Context, db *gorm.DB, feature *domain.Feature) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO features (
			id, org_id, code, name, description, feature_type, meter_id, active, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feature.ID,
		feature.OrgID,
		feature.Code,
		feature.Name,
		feature.Description,
		feature.Type,
		feature.MeterID,
		feature.Active,
		feature.Metadata,
		feature.CreatedAt,
		feature.UpdatedAt,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*domain.Feature, error) {
	var f domain.Feature
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, description, feature_type, meter_id, active, metadata, created_at, updated_at
		 FROM features WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&f).Error
	if err != nil {
		return nil, err
	}
	if f.ID == 0 {
		return nil, nil
	}
	return &f, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID int64, filter domain.ListRequest) ([]domain.Feature, error) {
	var items []domain.Feature
	stmt := db.WithContext(ctx).
		Model(&domain.Feature{}).
		Where("org_id = ?", orgID)

	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Code != "" {
		stmt = stmt.Where("code = ?", filter.Code)
	}
	if filter.FeatureType != nil {
		stmt = stmt.Where("feature_type = ?", *filter.FeatureType)
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
	}

	stmt = option.WithSortBy(option.WithQuerySortBy(filter.SortBy, filter.OrderBy, map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
	})).Apply(stmt)

	if err := stmt.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repo) ListByIDs(ctx context.Context, db *gorm.DB, orgID int64, ids []snowflake.ID) ([]domain.Feature, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []domain.Feature
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, description, feature_type, meter_id, active, metadata, created_at, updated_at
		 FROM features WHERE org_id = ? AND id IN ?`,
		orgID,
		ids,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repo) Update(ctx context.Context, db *gorm.DB, feature *domain.Feature) error {
	if feature == nil {
		return gorm.ErrInvalidData
	}
	return db.WithContext(ctx).Exec(
		`UPDATE features
		 SET name = ?, description = ?, feature_type = ?, meter_id = ?, active = ?, metadata = ?, updated_at = ?
		 WHERE org_id = ? AND id = ?`,
		feature.Name,
		feature.Description,
		feature.Type,
		feature.MeterID,
		feature.Active,
		feature.Metadata,
		feature.UpdatedAt,
		feature.OrgID,
		feature.ID,
	).Error
}
