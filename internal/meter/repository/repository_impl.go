package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	meterdomain "github.com/smallbiznis/railzway/internal/meter/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() meterdomain.Repository {
	return &repo{}
}

func (r *repo) Insert(ctx context.Context, db *gorm.DB, m *meterdomain.Meter) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO meters (id, org_id, code, name, aggregation, unit, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID,
		m.OrgID,
		m.Code,
		m.Name,
		m.Aggregation,
		m.Unit,
		m.Active,
		m.CreatedAt,
		m.UpdatedAt,
	).Error
}

func (r *repo) Update(ctx context.Context, db *gorm.DB, m *meterdomain.Meter) error {
	return db.WithContext(ctx).Exec(
		`UPDATE meters
		 SET name = ?, aggregation = ?, unit = ?, active = ?, updated_at = ?
		 WHERE org_id = ? AND id = ?`,
		m.Name,
		m.Aggregation,
		m.Unit,
		m.Active,
		m.UpdatedAt,
		m.OrgID,
		m.ID,
	).Error
}

func (r *repo) Delete(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) error {
	return db.WithContext(ctx).Exec(
		`DELETE FROM meters WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*meterdomain.Meter, error) {
	var meter meterdomain.Meter
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, aggregation, unit, active, created_at, updated_at
		 FROM meters WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&meter).Error
	if err != nil {
		return nil, err
	}
	if meter.ID == 0 {
		return nil, nil
	}
	return &meter, nil
}

func (r *repo) FindByCode(ctx context.Context, db *gorm.DB, orgID snowflake.ID, code string) (*meterdomain.Meter, error) {
	var meter meterdomain.Meter
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, aggregation, unit, active, created_at, updated_at
		 FROM meters WHERE org_id = ? AND code = ?`,
		orgID,
		code,
	).Scan(&meter).Error
	if err != nil {
		return nil, err
	}
	if meter.ID == 0 {
		return nil, nil
	}
	return &meter, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter meterdomain.ListRequest) ([]meterdomain.Meter, error) {
	var meters []meterdomain.Meter
	stmt := db.WithContext(ctx).
		Model(&meterdomain.Meter{}).
		Where("org_id = ?", orgID)

	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Code != "" {
		stmt = stmt.Where("code = ?", filter.Code)
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
	}

	stmt = option.WithSortBy(option.WithQuerySortBy(filter.SortBy, filter.OrderBy, map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"code":       true,
	})).Apply(stmt)

	if err := stmt.Find(&meters).Error; err != nil {
		return nil, err
	}
	return meters, nil
}
