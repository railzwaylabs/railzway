package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/smallbiznis/railzway/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("redis",
	fx.Provide(NewClient),
)

func NewClient(cfg config.Config) (*redis.Client, error) {
	// We can reuse RateLimit config for now as it contains Redis settings,
	// or ideally we should have a shared RedisConfig section.
	// Based on ratelimit/usage_ingest.go usage:
	redisCfg := cfg.RateLimit

	// If no address is provided, we might want to return an error or a nil client depending on strictness.
	// However, for the quota service, we need a client.
	// If RateLimit is disabled, we might still need Redis for Quotas if they are enabled.
	// Let's assume we use the address from RateLimit config for the shared Redis instance.
	
	// Fallback or default if needed, but for now let's stick to what's available.
	// If RedisAddr is empty, let's look for a generic one or error out if strict.
    // For this fixes, I will use the RateLimit config as the source of truth for Redis connection details 
    // to match existing patterns, but ensuring it's available even if rate limiting logic is disabled elsewhere.

	opt := &redis.Options{
		Addr:     redisCfg.RedisAddr,
		Password: redisCfg.RedisPassword,
		DB:       redisCfg.RedisDB,
	}

	client := redis.NewClient(opt)

	// Ping to verify connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
