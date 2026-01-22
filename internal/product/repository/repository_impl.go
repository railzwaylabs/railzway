package repository

import (
	"context"

	"github.com/smallbiznis/railzway/internal/product/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() domain.Repository {
	return &repo{}
}

func (r *repo) Create(ctx context.Context, db *gorm.DB, product *domain.Product) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO products (id, org_id, code, name, description, active, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		product.ID,
		product.OrgID,
		product.Code,
		product.Name,
		product.Description,
		product.Active,
		product.Metadata,
		product.CreatedAt,
		product.UpdatedAt,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*domain.Product, error) {
	var p domain.Product
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, description, active, metadata, created_at, updated_at
		 FROM products WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == 0 {
		return nil, nil
	}
	return &p, nil
}

func (r *repo) FindAll(ctx context.Context, db *gorm.DB, orgID int64) ([]domain.Product, error) {
	var items []domain.Product
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, code, name, description, active, metadata, created_at, updated_at
		 FROM products WHERE org_id = ? ORDER BY created_at ASC`,
		orgID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID int64, filter domain.ListRequest) ([]domain.Product, error) {
	var items []domain.Product
	stmt := db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("org_id = ?", orgID)

	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
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

func (r *repo) Update(ctx context.Context, db *gorm.DB, product *domain.Product) error {
	if product == nil {
		return gorm.ErrInvalidData
	}
	return db.WithContext(ctx).Exec(
		`UPDATE products
		 SET name = ?, description = ?, active = ?, metadata = ?, updated_at = ?
		 WHERE org_id = ? AND id = ?`,
		product.Name,
		product.Description,
		product.Active,
		product.Metadata,
		product.UpdatedAt,
		product.OrgID,
		product.ID,
	).Error
}
