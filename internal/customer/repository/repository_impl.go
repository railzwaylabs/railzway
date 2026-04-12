package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/customer/domain"
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

func (r *repository) Create(ctx context.Context, customer domain.Customer) error {
	return r.db.WithContext(ctx).Create(&customer).Error
}

func (r *repository) Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("id = ? AND org_id = ?", id, orgID).
		Updates(updates).Error
}

func (r *repository) FindByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, id).
		First(&customer).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *repository) FindByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&customer).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *repository) List(ctx context.Context, orgID uuid.UUID, filter domain.ListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Customer, error) {
	var customers []*domain.Customer
	stmt := r.db.WithContext(ctx).Model(&domain.Customer{}).Where("org_id = ?", orgID)

	if filter.Name != "" {
		stmt = stmt.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Email != "" {
		stmt = stmt.Where("email ILIKE ?", "%"+filter.Email+"%")
	}
	if filter.Currency != "" {
		stmt = stmt.Where("currency = ?", filter.Currency)
	}
	if filter.CreatedFrom != nil {
		stmt = stmt.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		stmt = stmt.Where("created_at <= ?", *filter.CreatedTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	err := stmt.
		Order("created_at desc, id desc").
		Limit(limit).
		Find(&customers).Error
	if err != nil {
		return nil, err
	}

	return customers, nil
}

func (r *repository) Count(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("org_id = ?", orgID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
