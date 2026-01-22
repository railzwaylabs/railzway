package local

import "go.uber.org/fx"

var Module = fx.Module("auth.local",
	fx.Provide(NewHandler),
	fx.Invoke(RegisterRoutes),
)
