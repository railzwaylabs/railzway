package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, meter *Meter) error
	Update(ctx context.Context, db *gorm.DB, meter *Meter) error
	Delete(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*Meter, error)
	FindByCode(ctx context.Context, db *gorm.DB, orgID snowflake.ID, code string) (*Meter, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter ListRequest) ([]Meter, error)
}
