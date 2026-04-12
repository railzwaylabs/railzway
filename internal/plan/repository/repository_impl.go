package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/plan/domain"
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

func (r *repository) CreatePlan(ctx context.Context, plan domain.Plan) error {
	return r.db.WithContext(ctx).Create(&plan).Error
}

func (r *repository) UpdatePlan(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Plan{}).
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(updates).Error
}

func (r *repository) FindPlanByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Plan, error) {
	var plan domain.Plan
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&plan).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (r *repository) FindPlanByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Plan, error) {
	var plan domain.Plan
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&plan).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (r *repository) ListPlans(ctx context.Context, orgID uuid.UUID, filter domain.PlanListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Plan, error) {
	var plans []*domain.Plan
	stmt := r.db.WithContext(ctx).Model(&domain.Plan{}).Where("org_id = ?", orgID)
	if filter.Code != "" {
		stmt = stmt.Where("code = ?", filter.Code)
	}
	if filter.Name != "" {
		stmt = stmt.Where("name = ?", filter.Name)
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

func (r *repository) CreatePlanPrice(ctx context.Context, price domain.PlanPrice) error {
	return r.db.WithContext(ctx).Create(&price).Error
}

func (r *repository) FindPlanPriceByID(ctx context.Context, orgID, id uuid.UUID) (*domain.PlanPrice, error) {
	var price domain.PlanPrice
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&price).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

func (r *repository) FindPlanPriceByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.PlanPrice, error) {
	var price domain.PlanPrice
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&price).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}

func (r *repository) ListPlanPrices(ctx context.Context, orgID uuid.UUID, filter domain.PlanPriceListFilter, limit int, cursor *domain.ListCursor) ([]*domain.PlanPrice, error) {
	var prices []*domain.PlanPrice
	stmt := r.db.WithContext(ctx).Model(&domain.PlanPrice{}).Where("org_id = ?", orgID)
	if filter.PlanID != uuid.Nil {
		stmt = stmt.Where("plan_id = ?", filter.PlanID)
	}
	if filter.PriceType != "" {
		stmt = stmt.Where("price_type = ?", filter.PriceType)
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
	}
	if filter.BillingInterval != "" {
		stmt = stmt.Where("billing_interval = ?", filter.BillingInterval)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&prices).Error
	if err != nil {
		return nil, err
	}
	return prices, nil
}

func (r *repository) CreatePlanAmount(ctx context.Context, amount domain.PlanAmount) error {
	return r.db.WithContext(ctx).Create(&amount).Error
}

func (r *repository) FindPlanAmountByID(ctx context.Context, orgID, id uuid.UUID) (*domain.PlanAmount, error) {
	var amount domain.PlanAmount
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&amount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &amount, nil
}

func (r *repository) FindPlanAmountByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.PlanAmount, error) {
	var amount domain.PlanAmount
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&amount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &amount, nil
}

func (r *repository) ListPlanAmounts(ctx context.Context, orgID uuid.UUID, filter domain.PlanAmountListFilter, limit int, cursor *domain.ListCursor) ([]*domain.PlanAmount, error) {
	var amounts []*domain.PlanAmount
	stmt := r.db.WithContext(ctx).Model(&domain.PlanAmount{}).Where("org_id = ?", orgID)
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if filter.Currency != "" {
		stmt = stmt.Where("currency = ?", filter.Currency)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&amounts).Error
	if err != nil {
		return nil, err
	}
	return amounts, nil
}

func (r *repository) CreatePlanTier(ctx context.Context, tier domain.PlanTier) error {
	return r.db.WithContext(ctx).Create(&tier).Error
}

func (r *repository) FindPlanTierByID(ctx context.Context, orgID, id uuid.UUID) (*domain.PlanTier, error) {
	var tier domain.PlanTier
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&tier).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}

func (r *repository) FindPlanTierByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.PlanTier, error) {
	var tier domain.PlanTier
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&tier).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}

func (r *repository) ListPlanTiers(ctx context.Context, orgID uuid.UUID, filter domain.PlanTierListFilter, limit int, cursor *domain.ListCursor) ([]*domain.PlanTier, error) {
	var tiers []*domain.PlanTier
	stmt := r.db.WithContext(ctx).Model(&domain.PlanTier{}).Where("org_id = ?", orgID)
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if filter.TierMode != "" {
		stmt = stmt.Where("tier_mode = ?", filter.TierMode)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&tiers).Error
	if err != nil {
		return nil, err
	}
	return tiers, nil
}

func (r *repository) ListPlanTiersByPrice(ctx context.Context, orgID, planPriceID uuid.UUID) ([]*domain.PlanTier, error) {
	var tiers []*domain.PlanTier
	err := r.db.WithContext(ctx).
		Model(&domain.PlanTier{}).
		Where("org_id = ? AND plan_price_id = ?", orgID, planPriceID).
		Order("start_quantity asc, id asc").
		Find(&tiers).Error
	if err != nil {
		return nil, err
	}
	return tiers, nil
}
