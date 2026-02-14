package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, meter *Meter) error
	Update(ctx context.Context, db *gorm.DB, meter *Meter) error
	Delete(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*Meter, error)
	FindByCode(ctx context.Context, db *gorm.DB, orgID snowflake.ID, code string) (*Meter, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID snowflake.ID, key string) (*Meter, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter ListRequest, page pagination.Pagination) ([]*Meter, error)
}
