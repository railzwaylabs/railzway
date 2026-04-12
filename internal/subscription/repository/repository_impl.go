package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/subscription/domain"
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

func (r *repository) CreateSubscription(ctx context.Context, sub domain.Subscription) error {
	return r.db.WithContext(ctx).Create(&sub).Error
}

func (r *repository) UpdateSubscription(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Subscription{}).
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(updates).Error
}

func (r *repository) FindSubscriptionByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Subscription, error) {
	var sub domain.Subscription
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) FindSubscriptionByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Subscription, error) {
	var sub domain.Subscription
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) ListSubscriptions(ctx context.Context, orgID uuid.UUID, filter domain.SubscriptionListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Subscription, error) {
	var subs []*domain.Subscription
	stmt := r.db.WithContext(ctx).Model(&domain.Subscription{}).Where("org_id = ?", orgID)
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.Status != "" {
		stmt = stmt.Where("status = ?", filter.Status)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *repository) CreateSubscriptionPeriod(ctx context.Context, period domain.SubscriptionPeriod) error {
	return r.db.WithContext(ctx).Create(&period).Error
}

func (r *repository) FindSubscriptionPeriodByTime(ctx context.Context, orgID, subscriptionID uuid.UUID, at time.Time) (*domain.SubscriptionPeriod, error) {
	var period domain.SubscriptionPeriod
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND subscription_id = ? AND period_start <= ? AND period_end >= ?", orgID, subscriptionID, at, at).
		Order("period_start DESC").
		First(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &period, nil
}

func (r *repository) FindSubscriptionPeriodForUpdate(ctx context.Context, orgID, periodID uuid.UUID) (*domain.SubscriptionPeriod, error) {
	var period domain.SubscriptionPeriod
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("org_id = ? AND id = ?", orgID, periodID).
		First(&period).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &period, nil
}

func (r *repository) ListOpenSubscriptionPeriods(ctx context.Context, asOf time.Time, limit int) ([]domain.SubscriptionPeriod, error) {
	var periods []domain.SubscriptionPeriod
	stmt := r.db.WithContext(ctx).Model(&domain.SubscriptionPeriod{}).
		Where("status = ? AND period_end <= ?", domain.PeriodStatusOpen, asOf).
		Order("period_end ASC")
	if limit > 0 {
		stmt = stmt.Limit(limit)
	}
	if err := stmt.Find(&periods).Error; err != nil {
		return nil, err
	}
	return periods, nil
}

func (r *repository) UpdateSubscriptionPeriod(ctx context.Context, orgID, periodID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.SubscriptionPeriod{}).
		Where("org_id = ? AND id = ?", orgID, periodID).
		Updates(updates).Error
}

func (r *repository) CreateSubscriptionItem(ctx context.Context, item domain.SubscriptionItem) error {
	return r.db.WithContext(ctx).Create(&item).Error
}

func (r *repository) FindSubscriptionItemByID(ctx context.Context, orgID, id uuid.UUID) (*domain.SubscriptionItem, error) {
	var item domain.SubscriptionItem
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *repository) FindSubscriptionItemByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.SubscriptionItem, error) {
	var item domain.SubscriptionItem
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *repository) ListSubscriptionItems(ctx context.Context, orgID uuid.UUID, filter domain.SubscriptionItemListFilter, limit int, cursor *domain.ListCursor) ([]*domain.SubscriptionItem, error) {
	var items []*domain.SubscriptionItem
	stmt := r.db.WithContext(ctx).Model(&domain.SubscriptionItem{}).Where("org_id = ?", orgID)
	if filter.SubscriptionID != uuid.Nil {
		stmt = stmt.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
