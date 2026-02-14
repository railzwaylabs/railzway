package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	"github.com/railzwaylabs/railzway/pkg/db/option"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() pricedomain.Repository {
	return &repo{}
}

func (r *repo) Insert(ctx context.Context, db *gorm.DB, p *pricedomain.Price) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO prices (
			id, org_id, product_id, code, name, description, idempotency_key,
			pricing_model, billing_mode, billing_interval, billing_interval_count,
			aggregate_usage, billing_unit, billing_threshold, tax_behavior, tax_code,
			version, is_default, active, retired_at, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID,
		p.OrgID,
		p.ProductID,
		p.Code,
		p.Name,
		p.Description,
		p.IdempotencyKey,
		p.PricingModel,
		p.BillingMode,
		p.BillingInterval,
		p.BillingIntervalCount,
		p.AggregateUsage,
		p.BillingUnit,
		p.BillingThreshold,
		p.TaxBehavior,
		p.TaxCode,
		p.Version,
		p.IsDefault,
		p.Active,
		p.RetiredAt,
		p.Metadata,
		p.CreatedAt,
		p.UpdatedAt,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*pricedomain.Price, error) {
	var p pricedomain.Price
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, product_id, code, name, description,
		 idempotency_key,
		 pricing_model, billing_mode, billing_interval, billing_interval_count,
		 aggregate_usage, billing_unit, billing_threshold, tax_behavior, tax_code,
		 version, is_default, active, retired_at, metadata, created_at, updated_at
		 FROM prices WHERE org_id = ? AND id = ?`,
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

func (r *repo) FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID snowflake.ID, key string) (*pricedomain.Price, error) {
	var p pricedomain.Price
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, product_id, code, name, description,
		 idempotency_key,
		 pricing_model, billing_mode, billing_interval, billing_interval_count,
		 aggregate_usage, billing_unit, billing_threshold, tax_behavior, tax_code,
		 version, is_default, active, retired_at, metadata, created_at, updated_at
		 FROM prices WHERE org_id = ? AND idempotency_key = ? LIMIT 1`,
		orgID,
		key,
	).Scan(&p).Error
	if err != nil {
		return nil, err
	}
	if p.ID == 0 {
		return nil, nil
	}
	return &p, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, opts pricedomain.ListOptions, page pagination.Pagination) ([]*pricedomain.Price, error) {
	var items []*pricedomain.Price

	query := db.WithContext(ctx).
		Model(&pricedomain.Price{}).
		Where("org_id = ?", orgID)

	if opts.ProductID != "" {
		query = query.Where("product_id = ?", opts.ProductID)
	}

	if opts.Code != nil {
		query = query.Where("code = ?", *opts.Code)
	}

	query = option.ApplyPagination(page).Apply(query)
	if page.PageToken != "" || page.PageSize > 0 {
		query = query.Order("created_at desc, id desc")
	} else {
		query = query.Order("created_at ASC")
	}

	err := query.Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}
