package repository

import (
	"context"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	"gorm.io/gorm"
)

type checkoutSessionRepo struct {
	db *gorm.DB
}

func NewCheckoutSessionRepository(db *gorm.DB) domain.CheckoutSessionRepository {
	return &checkoutSessionRepo{
		db: db,
	}
}

func (r *checkoutSessionRepo) Insert(ctx context.Context, db *gorm.DB, session *domain.CheckoutSession) error {
	if db == nil {
		db = r.db
	}
	return db.WithContext(ctx).Create(session).Error
}

func (r *checkoutSessionRepo) Update(ctx context.Context, db *gorm.DB, session *domain.CheckoutSession) error {
	if db == nil {
		db = r.db
	}
	return db.WithContext(ctx).Save(session).Error
}

func (r *checkoutSessionRepo) FindByID(ctx context.Context, db *gorm.DB, id snowflake.ID) (*domain.CheckoutSession, error) {
	if db == nil {
		db = r.db
	}
	var session domain.CheckoutSession
	if err := db.WithContext(ctx).First(&session, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &session, nil
}

func (r *checkoutSessionRepo) FindByProviderSessionID(ctx context.Context, db *gorm.DB, provider, providerSessionID string) (*domain.CheckoutSession, error) {
	if db == nil {
		db = r.db
	}
	var session domain.CheckoutSession
	if err := db.WithContext(ctx).
		Where("provider = ? AND provider_session_id = ?", provider, providerSessionID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *checkoutSessionRepo) FindByAnyProviderSessionID(ctx context.Context, db *gorm.DB, providerSessionID string) (*domain.CheckoutSession, error) {
	if db == nil {
		db = r.db
	}
	var session domain.CheckoutSession
	if err := db.WithContext(ctx).
		Where("provider_session_id = ?", providerSessionID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}
