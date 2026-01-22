package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, tier *PriceTier) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*PriceTier, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]PriceTier, error)
}
