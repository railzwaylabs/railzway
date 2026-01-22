package repository

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/productfeature/domain"
	"gorm.io/gorm"
)

type repo struct{}

func Provide() domain.Repository {
	return &repo{}
}

func (r *repo) ListByProduct(ctx context.Context, db *gorm.DB, orgID, productID snowflake.ID) ([]domain.FeatureAssignment, error) {
	var items []domain.FeatureAssignment
	err := db.WithContext(ctx).Raw(
		`SELECT pf.product_id, pf.feature_id, pf.created_at,
				f.code, f.name, f.feature_type, f.meter_id, f.active
		   FROM product_features pf
		   JOIN products p ON p.id = pf.product_id AND p.org_id = ?
		   JOIN features f ON f.id = pf.feature_id AND f.org_id = ?
		  WHERE pf.product_id = ?
		  ORDER BY pf.created_at ASC`,
		orgID,
		orgID,
		productID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repo) ListByProducts(ctx context.Context, db *gorm.DB, orgID snowflake.ID, productIDs []snowflake.ID) ([]domain.FeatureAssignment, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var items []domain.FeatureAssignment
	err := db.WithContext(ctx).Raw(
		`SELECT pf.product_id, pf.feature_id, pf.created_at,
				f.code, f.name, f.feature_type, f.meter_id, f.active
		   FROM product_features pf
		   JOIN products p ON p.id = pf.product_id AND p.org_id = ?
		   JOIN features f ON f.id = pf.feature_id AND f.org_id = ?
		  WHERE pf.product_id IN ?
		  ORDER BY pf.created_at ASC`,
		orgID,
		orgID,
		productIDs,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repo) Replace(ctx context.Context, db *gorm.DB, productID snowflake.ID, featureIDs []snowflake.ID, now time.Time) error {
	if err := db.WithContext(ctx).Exec(
		`DELETE FROM product_features WHERE product_id = ?`,
		productID,
	).Error; err != nil {
		return err
	}

	for _, featureID := range featureIDs {
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO product_features (product_id, feature_id, created_at)
			 VALUES (?, ?, ?)`,
			productID,
			featureID,
			now,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
