package bootstrap

import (
	"context"

	"go.uber.org/fx"
)

// EnforceSchemaGate fails fast during application startup when the schema is not active.
func EnforceSchemaGate(lc fx.Lifecycle, gate SchemaGate) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return gate.MustBeActive(ctx)
		},
	})
}
