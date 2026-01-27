package bootstrap

import "go.uber.org/fx"

var Module = fx.Module("bootstrap",
	fx.Provide(NewSchemaGate),
	fx.Provide(NewOrgStateService),
	fx.Provide(NewOrgGate),
)
