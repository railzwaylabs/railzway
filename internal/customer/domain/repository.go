package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, customer *Customer) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*Customer, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID snowflake.ID, key string) (*Customer, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter ListCustomerFilter, page pagination.Pagination) ([]*Customer, error)
	Count(ctx context.Context, db *gorm.DB, orgID snowflake.ID) (int64, error)
}
