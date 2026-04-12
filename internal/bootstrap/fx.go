package bootstrap

import "go.uber.org/fx"

var Module = fx.Module("bootstrap",
	fx.Invoke(EnsureSeed),
)
