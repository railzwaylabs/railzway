package reference

import "go.uber.org/fx"

var Module = fx.Module("reference.repository",
	fx.Provide(NewRepository),
)
