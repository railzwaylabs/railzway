package liveevents

import "go.uber.org/fx"

var Module = fx.Module("usage.liveevents",
	fx.Provide(NewHub),
)
