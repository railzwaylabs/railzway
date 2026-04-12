package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/rating/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *repository) CreateRatingResult(ctx context.Context, result domain.RatingResult) error {
	return r.db.WithContext(ctx).Create(&result).Error
}

func (r *repository) FindRatingByUsageEvent(ctx context.Context, orgID, usageEventID uuid.UUID) (*domain.RatingResult, error) {
	var res domain.RatingResult
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND usage_event_id = ?", orgID, usageEventID).
		First(&res).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *repository) ListRatingResults(ctx context.Context, orgID uuid.UUID, filter domain.RatingResultFilter, limit int, cursor *domain.ListCursor) ([]*domain.RatingResult, error) {
	var results []*domain.RatingResult
	stmt := r.db.WithContext(ctx).Model(&domain.RatingResult{}).Where("org_id = ?", orgID)
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.SubscriptionID != uuid.Nil {
		stmt = stmt.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if filter.MeterID != uuid.Nil {
		stmt = stmt.Where("meter_id = ?", filter.MeterID)
	}
	if filter.UsageEventID != uuid.Nil {
		stmt = stmt.Where("usage_event_id = ?", filter.UsageEventID)
	}
	if filter.WindowStartFrom != nil {
		stmt = stmt.Where("window_start >= ?", *filter.WindowStartFrom)
	}
	if filter.WindowStartTo != nil {
		stmt = stmt.Where("window_start <= ?", *filter.WindowStartTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *repository) UpsertUsageAggregate(ctx context.Context, aggregate domain.UsageAggregate) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO usage_aggregates (
			id, org_id, customer_id, subscription_id, plan_price_id, plan_amount_id, meter_id, currency,
			period_start, period_end, quantity, amount_cents, last_event_at, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, customer_id, plan_price_id, plan_amount_id, meter_id, period_start, period_end)
		DO UPDATE SET quantity = EXCLUDED.quantity,
		              amount_cents = EXCLUDED.amount_cents,
		              last_event_at = EXCLUDED.last_event_at,
		              updated_at = EXCLUDED.updated_at`,
		aggregate.ID,
		aggregate.OrgID,
		aggregate.CustomerID,
		aggregate.SubscriptionID,
		aggregate.PlanPriceID,
		aggregate.PlanAmountID,
		aggregate.MeterID,
		aggregate.Currency,
		aggregate.PeriodStart,
		aggregate.PeriodEnd,
		aggregate.Quantity,
		aggregate.AmountCents,
		aggregate.LastEventAt,
		aggregate.Metadata,
		aggregate.CreatedAt,
		aggregate.UpdatedAt,
	).Error
}

func (r *repository) CreateUsageAggregate(ctx context.Context, aggregate domain.UsageAggregate) error {
	return r.db.WithContext(ctx).Create(&aggregate).Error
}

func (r *repository) UpdateUsageAggregate(ctx context.Context, aggregate domain.UsageAggregate) error {
	return r.db.WithContext(ctx).
		Model(&domain.UsageAggregate{}).
		Where("id = ?", aggregate.ID).
		Updates(map[string]interface{}{
			"quantity":      aggregate.Quantity,
			"amount_cents":  aggregate.AmountCents,
			"last_event_at": aggregate.LastEventAt,
			"metadata":      aggregate.Metadata,
			"updated_at":    aggregate.UpdatedAt,
		}).Error
}

func (r *repository) GetUsageAggregate(ctx context.Context, orgID, customerID, planPriceID, planAmountID, meterID uuid.UUID, periodStart, periodEnd time.Time) (*domain.UsageAggregate, error) {
	var agg domain.UsageAggregate
	err := r.db.WithContext(ctx).
		Where(
			"org_id = ? AND customer_id = ? AND plan_price_id = ? AND plan_amount_id = ? AND meter_id = ? AND period_start = ? AND period_end = ?",
			orgID, customerID, planPriceID, planAmountID, meterID, periodStart, periodEnd,
		).
		First(&agg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &agg, nil
}

func (r *repository) GetUsageAggregateForUpdate(ctx context.Context, orgID, customerID, planPriceID, planAmountID, meterID uuid.UUID, periodStart, periodEnd time.Time) (*domain.UsageAggregate, error) {
	var agg domain.UsageAggregate
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"org_id = ? AND customer_id = ? AND plan_price_id = ? AND plan_amount_id = ? AND meter_id = ? AND period_start = ? AND period_end = ?",
			orgID, customerID, planPriceID, planAmountID, meterID, periodStart, periodEnd,
		).
		First(&agg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &agg, nil
}

func (r *repository) ListUsageAggregates(ctx context.Context, orgID uuid.UUID, filter domain.UsageAggregateFilter, limit int, cursor *domain.ListCursor) ([]*domain.UsageAggregate, error) {
	var results []*domain.UsageAggregate
	stmt := r.db.WithContext(ctx).Model(&domain.UsageAggregate{}).Where("org_id = ?", orgID)
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.SubscriptionID != uuid.Nil {
		stmt = stmt.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if filter.MeterID != uuid.Nil {
		stmt = stmt.Where("meter_id = ?", filter.MeterID)
	}
	if filter.PeriodStartFrom != nil {
		stmt = stmt.Where("period_start >= ?", *filter.PeriodStartFrom)
	}
	if filter.PeriodStartTo != nil {
		stmt = stmt.Where("period_start <= ?", *filter.PeriodStartTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}
