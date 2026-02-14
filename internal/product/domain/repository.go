package domain

import (
	"context"

	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, product *Product) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*Product, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID int64, key string) (*Product, error)
	FindAll(ctx context.Context, db *gorm.DB, orgID int64) ([]Product, error)
	List(ctx context.Context, db *gorm.DB, orgID int64, filter ListRequest, page pagination.Pagination) ([]*Product, error)
	Update(ctx context.Context, db *gorm.DB, product *Product) error
}
