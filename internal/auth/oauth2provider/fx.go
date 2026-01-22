package oauth2provider

import "go.uber.org/fx"

var Module = fx.Module("auth.oauth2.provider",
	fx.Provide(NewConfig),
	fx.Provide(NewStore),
	fx.Provide(NewService),
	fx.Provide(NewHandler),
	fx.Invoke(RegisterRoutes),
)
