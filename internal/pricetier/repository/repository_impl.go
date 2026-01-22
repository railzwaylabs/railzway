package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	pricetierdomain "github.com/smallbiznis/railzway/internal/pricetier/domain"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() pricetierdomain.Repository {
	return &repo{}
}

func (r *repo) Insert(ctx context.Context, db *gorm.DB, tier *pricetierdomain.PriceTier) error {
	return db.WithContext(ctx).Exec(
		`INSERT INTO price_tiers (
			id, org_id, price_id, tier_mode, start_quantity, end_quantity, unit_amount_cents,
			flat_amount_cents, unit, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tier.ID,
		tier.OrgID,
		tier.PriceID,
		tier.TierMode,
		tier.StartQuantity,
		tier.EndQuantity,
		tier.UnitAmountCents,
		tier.FlatAmountCents,
		tier.Unit,
		tier.Metadata,
		tier.CreatedAt,
		tier.UpdatedAt,
	).Error
}

func (r *repo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*pricetierdomain.PriceTier, error) {
	var tier pricetierdomain.PriceTier
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, price_id, tier_mode, start_quantity, end_quantity, unit_amount_cents,
		 flat_amount_cents, unit, metadata, created_at, updated_at
		 FROM price_tiers WHERE org_id = ? AND id = ?`,
		orgID,
		id,
	).Scan(&tier).Error
	if err != nil {
		return nil, err
	}
	if tier.ID == 0 {
		return nil, nil
	}
	return &tier, nil
}

func (r *repo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]pricetierdomain.PriceTier, error) {
	var items []pricetierdomain.PriceTier
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, price_id, tier_mode, start_quantity, end_quantity, unit_amount_cents,
		 flat_amount_cents, unit, metadata, created_at, updated_at
		 FROM price_tiers WHERE org_id = ? ORDER BY created_at ASC`,
		orgID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
