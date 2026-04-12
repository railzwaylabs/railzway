package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/productfeature/domain"
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

func (r *repository) ListByProduct(ctx context.Context, orgID, productID uuid.UUID) ([]domain.FeatureAssignment, error) {
	var items []domain.FeatureAssignment
	err := r.db.WithContext(ctx).Raw(
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

func (r *repository) ListByProducts(ctx context.Context, orgID uuid.UUID, productIDs []uuid.UUID) ([]domain.FeatureAssignment, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var items []domain.FeatureAssignment
	err := r.db.WithContext(ctx).Raw(
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

func (r *repository) Replace(ctx context.Context, productID uuid.UUID, featureIDs []uuid.UUID, now time.Time) error {
	if err := r.db.WithContext(ctx).Exec(
		`DELETE FROM product_features WHERE product_id = ?`,
		productID,
	).Error; err != nil {
		return err
	}

	for _, featureID := range featureIDs {
		if err := r.db.WithContext(ctx).Exec(
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
