package ratelimit

import (
	"fmt"

	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/redis"
	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Module wires the rate limiter.
var Module = fx.Module("ratelimit",
	fx.Provide(
		fx.Annotate(
			func(cfg *config.Config, client *redisv9.Client) *Limiter {
				rlCfg := Config{
					WindowSec:                              cfg.RateLimit.UsageEventsWindowSec,
					UsageEventsSubscriptionPerMin:          cfg.RateLimit.UsageEventsSubscriptionPerMin,
					UsageEventsCustomerPerMin:              cfg.RateLimit.UsageEventsCustomerPerMin,
					UsageEventsOrgPerMin:                   cfg.RateLimit.UsageEventsOrgPerMin,
					UsageEventsConcurrencyPerCustomerMeter: cfg.RateLimit.UsageEventsConcurrencyPerCustomerMeter,
					UsageEventsConcurrencyTTLSeconds:       cfg.RateLimit.UsageEventsConcurrencyTTLSeconds,
				}
				return NewLimiter(client, rlCfg)
			},
			fx.ParamTags("", fmt.Sprintf(`name:"%s"`, redis.StoreClientName)),
		),
	),
)
