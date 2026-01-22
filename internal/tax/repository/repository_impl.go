package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	taxdomain "github.com/smallbiznis/railzway/internal/tax/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) taxdomain.Repository {
	return &repository{db: db}
}

func (r *repository) GetActiveTaxDefinition(ctx context.Context, orgID snowflake.ID) (*taxdomain.TaxDefinition, error) {
	var def taxdomain.TaxDefinition
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, org_id, name, code, tax_mode, rate, description, is_enabled, created_at, updated_at
		 FROM tax_definitions
		 WHERE org_id = ? AND is_enabled = true
		 ORDER BY id ASC
		 LIMIT 1`,
		orgID,
	).Scan(&def).Error
	if err != nil {
		return nil, err
	}
	if def.ID == 0 {
		return nil, nil
	}
	return &def, nil
}

func (r *repository) Create(ctx context.Context, def *taxdomain.TaxDefinition) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO tax_definitions (
			id, org_id, name, code, tax_mode, rate, description, is_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		def.ID,
		def.OrgID,
		def.Name,
		def.Code,
		def.TaxMode,
		def.Rate,
		def.Description,
		def.IsEnabled,
		def.CreatedAt,
		def.UpdatedAt,
	).Error
}

func (r *repository) FindByID(ctx context.Context, orgID, id snowflake.ID) (*taxdomain.TaxDefinition, error) {
	var def taxdomain.TaxDefinition
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, org_id, name, code, tax_mode, rate, description, is_enabled, created_at, updated_at
		 FROM tax_definitions
		 WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&def).Error
	if err != nil {
		return nil, err
	}
	if def.ID == 0 {
		return nil, nil
	}
	return &def, nil
}

func (r *repository) List(ctx context.Context, orgID snowflake.ID, filter taxdomain.ListRequest) ([]taxdomain.TaxDefinition, error) {
	var items []taxdomain.TaxDefinition
	stmt := r.db.WithContext(ctx).
		Model(&taxdomain.TaxDefinition{}).
		Where("org_id = ?", orgID)

	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Code != "" {
		stmt = stmt.Where("code = ?", filter.Code)
	}
	if filter.IsEnabled != nil {
		stmt = stmt.Where("is_enabled = ?", *filter.IsEnabled)
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

func (r *repository) Update(ctx context.Context, def *taxdomain.TaxDefinition) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE tax_definitions
		 SET name = ?, tax_mode = ?, rate = ?, description = ?, is_enabled = ?, updated_at = ?
		 WHERE org_id = ? AND id = ?`,
		def.Name,
		def.TaxMode,
		def.Rate,
		def.Description,
		def.IsEnabled,
		def.UpdatedAt,
		def.OrgID,
		def.ID,
	).Error
}
