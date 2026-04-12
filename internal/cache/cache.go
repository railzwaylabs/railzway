package cache

import (
	"context"
	"fmt"

	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// ClientName is the Fx name for the Redis client used for caching.
const ClientName = "redis_cache"

func Register() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				func(cfg *config.Config) (*redis.Client, error) {
					return redis.NewClient(&redis.Options{
						Addr:     cfg.CacheConfig.CacheURL,
						Username: cfg.CacheConfig.CacheUsername,
						Password: cfg.CacheConfig.CachePassword,
						DB:       cfg.CacheConfig.CacheDB,
					}), nil
				},
				fx.ResultTags(fmt.Sprintf(`name:"%s"`, ClientName)),
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
				fx.ParamTags("", fmt.Sprintf(`name:"%s"`, ClientName)),
			),
		),
	)
}
