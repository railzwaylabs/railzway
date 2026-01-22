package repository

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	ratingdomain "github.com/smallbiznis/railzway/internal/rating/domain"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	usagedomain "github.com/smallbiznis/railzway/internal/usage/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) ratingdomain.Repository {
	return &repository{db: db}
}

func (r *repository) GetBillingCycle(ctx context.Context, id snowflake.ID) (*ratingdomain.BillingCycleRow, error) {
	var row ratingdomain.BillingCycleRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, org_id, subscription_id, period_start, period_end, status
		 FROM billing_cycles
		 WHERE id = ?`,
		id,
	).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *repository) GetSubscription(ctx context.Context, orgID, subID snowflake.ID) (*subscriptiondomain.Subscription, error) {
	var sub subscriptiondomain.Subscription
	err := r.db.WithContext(ctx).Model(&subscriptiondomain.Subscription{}).
		Where("org_id = ? AND id = ?", orgID, subID).
		First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) ListSubscriptionItems(ctx context.Context, orgID, subID snowflake.ID) ([]ratingdomain.SubscriptionItemRow, error) {
	var items []ratingdomain.SubscriptionItemRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, org_id, subscription_id, price_id, meter_id
		 FROM subscription_items
		 WHERE org_id = ? AND subscription_id = ?`,
		orgID,
		subID,
	).Scan(&items).Error
	return items, err
}

func (r *repository) ListEntitlements(ctx context.Context, orgID, subID snowflake.ID, start, end time.Time) ([]subscriptiondomain.SubscriptionEntitlement, error) {
	var rows []subscriptiondomain.SubscriptionEntitlement
	err := r.db.WithContext(ctx).Raw(`
		SELECT * FROM subscription_entitlements
		WHERE org_id = ? AND subscription_id = ?
		AND effective_from < ?
		AND (effective_to IS NULL OR effective_to > ?)
	`, orgID, subID, end, start).Scan(&rows).Error
	return rows, err
}

func (r *repository) AggregateUsage(ctx context.Context, orgID, subID, meterID snowflake.ID, start, end time.Time) (float64, error) {
	var quantity float64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(value), 0)
		 FROM usage_events
		 WHERE org_id = ? AND subscription_id = ? AND meter_id = ?
		 AND recorded_at >= ? AND recorded_at < ? AND status = ?`,
		orgID,
		subID,
		meterID,
		start,
		end,
		usagedomain.UsageStatusEnriched,
	).Scan(&quantity).Error
	return quantity, err
}

func (r *repository) DeleteRatingResults(ctx context.Context, cycleID snowflake.ID) error {
	return r.db.WithContext(ctx).Where("billing_cycle_id = ?", cycleID).Delete(&ratingdomain.RatingResult{}).Error
}

func (r *repository) InsertRatingResult(ctx context.Context, result ratingdomain.RatingResult) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO rating_results (
			id, org_id, subscription_id, billing_cycle_id, meter_id, price_id, feature_code,
			quantity, unit_price, amount, currency, period_start, period_end,
			source, checksum, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (checksum) DO NOTHING`,
		result.ID,
		result.OrgID,
		result.SubscriptionID,
		result.BillingCycleID,
		result.MeterID,
		result.PriceID,
		result.FeatureCode,
		result.Quantity,
		result.UnitPrice,
		result.Amount,
		result.Currency,
		result.PeriodStart,
		result.PeriodEnd,
		result.Source,
		result.Checksum,
		result.CreatedAt,
	).Error
}
