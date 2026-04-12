package redis

import (
	"context"
	"fmt"

	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// StoreClientName is the Fx name for the Redis client used for sessions/idempotency.
const StoreClientName = "redis_store"

func Register() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(cfg *config.Config) (*redis.Client, error) {
					return redis.NewClient(&redis.Options{
						Addr:     cfg.RedisConfig.RedisURL,
						Username: cfg.RedisConfig.RedisUsername,
						Password: cfg.RedisConfig.RedisPassword,
						DB:       cfg.RedisConfig.RedisDB,
					}), nil
				},
				fx.ResultTags(fmt.Sprintf(`name:"%s"`, StoreClientName)),
			),
		),
		fx.Invoke(
			fx.Annotate(
				func(lc fx.Lifecycle, client *redis.Client) {
					lc.Append(fx.Hook{
						OnStop: func(ctx context.Context) error {
							return client.Close()
						},
					})
				},
				fx.ParamTags("", fmt.Sprintf(`name:"%s"`, StoreClientName)),
			),
		),
	)
}
