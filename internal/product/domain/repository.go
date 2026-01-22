package domain

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, product *Product) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*Product, error)
	FindAll(ctx context.Context, db *gorm.DB, orgID int64) ([]Product, error)
	List(ctx context.Context, db *gorm.DB, orgID int64, filter ListRequest) ([]Product, error)
	Update(ctx context.Context, db *gorm.DB, product *Product) error
}
