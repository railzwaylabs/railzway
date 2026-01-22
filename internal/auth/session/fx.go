package session

import "go.uber.org/fx"

var Module = fx.Module("auth.session",
	fx.Provide(NewManager),
)
