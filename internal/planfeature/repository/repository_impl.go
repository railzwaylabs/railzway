package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/planfeature/domain"
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

func (r *repository) ListByPlan(ctx context.Context, orgID, planID uuid.UUID) ([]domain.FeatureAssignment, error) {
	var items []domain.FeatureAssignment
	err := r.db.WithContext(ctx).Raw(
		`SELECT pf.plan_id, pf.feature_id, pf.enabled, pf.limit_numeric, pf.limit_unit, pf.reset_period, pf.created_at,
				f.code, f.name, f.feature_type, f.meter_id, f.active
		   FROM plan_features pf
		   JOIN plans p ON p.id = pf.plan_id AND p.org_id = ?
		   JOIN features f ON f.id = pf.feature_id AND f.org_id = ?
		  WHERE pf.org_id = ? AND pf.plan_id = ?
		  ORDER BY pf.created_at ASC`,
		orgID,
		orgID,
		orgID,
		planID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) ListByPlans(ctx context.Context, orgID uuid.UUID, planIDs []uuid.UUID) ([]domain.FeatureAssignment, error) {
	if len(planIDs) == 0 {
		return nil, nil
	}
	var items []domain.FeatureAssignment
	err := r.db.WithContext(ctx).Raw(
		`SELECT pf.plan_id, pf.feature_id, pf.enabled, pf.limit_numeric, pf.limit_unit, pf.reset_period, pf.created_at,
				f.code, f.name, f.feature_type, f.meter_id, f.active
		   FROM plan_features pf
		   JOIN plans p ON p.id = pf.plan_id AND p.org_id = ?
		   JOIN features f ON f.id = pf.feature_id AND f.org_id = ?
		  WHERE pf.org_id = ? AND pf.plan_id IN ?
		  ORDER BY pf.created_at ASC`,
		orgID,
		orgID,
		orgID,
		planIDs,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) Replace(ctx context.Context, orgID, planID uuid.UUID, items []domain.PlanFeature, now time.Time) error {
	if err := r.db.WithContext(ctx).Exec(
		`DELETE FROM plan_features WHERE org_id = ? AND plan_id = ?`,
		orgID,
		planID,
	).Error; err != nil {
		return err
	}

	for _, item := range items {
		item.CreatedAt = now
		item.UpdatedAt = now
		if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}
