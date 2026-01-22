package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
)

type Repository interface {
	GetActiveTaxDefinition(ctx context.Context, orgID snowflake.ID) (*TaxDefinition, error)
	Create(ctx context.Context, def *TaxDefinition) error
	FindByID(ctx context.Context, orgID, id snowflake.ID) (*TaxDefinition, error)
	List(ctx context.Context, orgID snowflake.ID, filter ListRequest) ([]TaxDefinition, error)
	Update(ctx context.Context, def *TaxDefinition) error
}
