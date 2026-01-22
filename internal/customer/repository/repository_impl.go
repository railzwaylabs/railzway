package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/customer/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"github.com/smallbiznis/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() domain.Repository {
	return &repo{}
}

func (r *repo) Insert(ctx context.Context, db *gorm.DB, customer *domain.Customer) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO customers (id, org_id, name, email, currency, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		customer.ID,
		customer.OrgID,
		customer.Name,
		customer.Email,
		customer.Currency,
		customer.Metadata,
		customer.CreatedAt,
		customer.UpdatedAt,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*domain.Customer, error) {
	var customer domain.Customer
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, name, email, currency, metadata, created_at, updated_at
		 FROM customers WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&customer).Error
	if err != nil {
		return nil, err
	}
	if customer.ID == 0 {
		return nil, nil
	}
	return &customer, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter domain.ListCustomerFilter, page pagination.Pagination) ([]*domain.Customer, error) {
	var customers []*domain.Customer
	stmt := db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("org_id = ?", orgID)
	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Email != "" {
		stmt = stmt.Where("email = ?", filter.Email)
	}
	if filter.Currency != "" {
		stmt = stmt.Where("currency = ?", filter.Currency)
	}
	if filter.CreatedFrom != nil {
		stmt = stmt.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		stmt = stmt.Where("created_at <= ?", *filter.CreatedTo)
	}
	stmt = option.ApplyPagination(page).Apply(stmt)
	err := stmt.
		Order("created_at desc, id desc").
		Find(&customers).Error
	if err != nil {
		return nil, err
	}
	return customers, nil
}

func (r *repo) Count(ctx context.Context, db *gorm.DB, orgID snowflake.ID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("org_id = ?", orgID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
