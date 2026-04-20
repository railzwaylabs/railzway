package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/coupon/domain"
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

func (r *repository) CreateCoupon(ctx context.Context, coupon domain.Coupon) error {
	return r.db.WithContext(ctx).Create(&coupon).Error
}

func (r *repository) GetCoupon(ctx context.Context, orgID, id uuid.UUID) (*domain.Coupon, error) {
	var coupon domain.Coupon
	err := r.db.WithContext(ctx).Where("org_id = ? AND id = ?", orgID, id).First(&coupon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &coupon, nil
}

func (r *repository) ListCoupons(ctx context.Context, orgID uuid.UUID) ([]domain.Coupon, error) {
	var coupons []domain.Coupon
	err := r.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&coupons).Error
	return coupons, err
}

func (r *repository) CreatePromotionCode(ctx context.Context, promo domain.PromotionCode) error {
	return r.db.WithContext(ctx).Create(&promo).Error
}

func (r *repository) GetPromotionCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.PromotionCode, error) {
	var promo domain.PromotionCode
	err := r.db.WithContext(ctx).Where("org_id = ? AND code = ?", orgID, code).First(&promo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &promo, nil
}

func (r *repository) IncrementRedemptionCount(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.PromotionCode{}).
		Where("org_id = ? AND id = ?", orgID, id).
		Update("redemption_count", gorm.Expr("redemption_count + 1")).Error
}

func (r *repository) ApplyCoupon(ctx context.Context, subCoupon domain.SubscriptionCoupon) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "org_id"},
				{Name: "subscription_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"coupon_id", "applied_at"}),
		}).
		Create(&subCoupon).Error
}

func (r *repository) GetSubscriptionCoupon(ctx context.Context, orgID, subID uuid.UUID) (*domain.SubscriptionCoupon, error) {
	var subCoupon domain.SubscriptionCoupon
	err := r.db.WithContext(ctx).Where("org_id = ? AND subscription_id = ?", orgID, subID).First(&subCoupon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &subCoupon, nil
}
