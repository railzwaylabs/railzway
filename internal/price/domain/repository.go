package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type ListOptions struct {
	ProductID string  `form:"product_id"`
	Code      *string `form:"code"`
	PageToken string  `form:"page_token"`
	PageSize  int32   `form:"page_size"`
}

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, price *Price) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*Price, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID snowflake.ID, key string) (*Price, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, opts ListOptions, page pagination.Pagination) ([]*Price, error)
}
