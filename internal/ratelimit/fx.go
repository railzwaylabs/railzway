package ratelimit

import "go.uber.org/fx"

var Module = fx.Module("rate.limit",
	fx.Provide(NewUsageIngestLimiter),
)
