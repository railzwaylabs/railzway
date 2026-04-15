package ratelimit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type UsageEventsKey struct {
	OrgID          string
	CustomerID     string
	SubscriptionID string
	MeterCode      string
}

type Decision struct {
	Allowed       bool
	Limit         int
	Remaining     int
	ResetSeconds  int
	Scope         string
	Reason        string
	LimitDisabled bool
}

type Limiter struct {
	client        *redis.Client
	window        time.Duration
	subLimit      int
	customerLimit int
	orgLimit      int
	concurrency   int
	concurrencyTTL time.Duration
}

func NewLimiter(client *redis.Client, cfg Config) *Limiter {
	if cfg.WindowSec <= 0 {
		cfg.WindowSec = 60
	}
	if cfg.UsageEventsConcurrencyTTLSeconds <= 0 {
		cfg.UsageEventsConcurrencyTTLSeconds = 5
	}
	return &Limiter{
		client:         client,
		window:         time.Duration(cfg.WindowSec) * time.Second,
		subLimit:       cfg.UsageEventsSubscriptionPerMin,
		customerLimit:  cfg.UsageEventsCustomerPerMin,
		orgLimit:       cfg.UsageEventsOrgPerMin,
		concurrency:    cfg.UsageEventsConcurrencyPerCustomerMeter,
		concurrencyTTL: time.Duration(cfg.UsageEventsConcurrencyTTLSeconds) * time.Second,
	}
}

func (l *Limiter) AcquireUsageConcurrency(ctx context.Context, key UsageEventsKey) (func(), Decision, error) {
	noRelease := func() {}
	if l == nil || l.client == nil || l.concurrency <= 0 {
		return noRelease, Decision{Allowed: true, LimitDisabled: true, Scope: "concurrency"}, nil
	}
	if strings.TrimSpace(key.CustomerID) == "" || strings.TrimSpace(key.MeterCode) == "" {
		return noRelease, Decision{Allowed: true, LimitDisabled: true, Scope: "concurrency"}, nil
	}
	meterCode := strings.ToLower(strings.TrimSpace(key.MeterCode))
	redisKey := fmt.Sprintf("rl:usage_events:conc:%s:%s:%s", key.OrgID, key.CustomerID, meterCode)

	if l.concurrency == 1 {
		ok, err := l.client.SetNX(ctx, redisKey, "1", l.concurrencyTTL).Result()
		if err != nil {
			return noRelease, Decision{}, err
		}
		if !ok {
			return noRelease, Decision{
				Allowed:      false,
				Limit:        l.concurrency,
				Remaining:    0,
				ResetSeconds: int(l.concurrencyTTL.Seconds()),
				Scope:        "concurrency",
				Reason:       "concurrency_limit",
			}, nil
		}
		release := func() {
			_ = l.client.Del(context.Background(), redisKey).Err()
		}
		return release, Decision{
			Allowed:      true,
			Limit:        l.concurrency,
			Remaining:    0,
			ResetSeconds: int(l.concurrencyTTL.Seconds()),
			Scope:        "concurrency",
		}, nil
	}

	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return noRelease, Decision{}, err
	}
	if count == 1 {
		_, _ = l.client.Expire(ctx, redisKey, l.concurrencyTTL).Result()
	}
	if count > int64(l.concurrency) {
		_, _ = l.client.Decr(ctx, redisKey).Result()
		return noRelease, Decision{
			Allowed:      false,
			Limit:        l.concurrency,
			Remaining:    0,
			ResetSeconds: int(l.concurrencyTTL.Seconds()),
			Scope:        "concurrency",
			Reason:       "concurrency_limit",
		}, nil
	}
	release := func() {
		_, _ = l.client.Decr(context.Background(), redisKey).Result()
	}
	remaining := l.concurrency - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return release, Decision{
		Allowed:      true,
		Limit:        l.concurrency,
		Remaining:    remaining,
		ResetSeconds: int(l.concurrencyTTL.Seconds()),
		Scope:        "concurrency",
	}, nil
}

func (l *Limiter) AllowUsageEvents(ctx context.Context, key UsageEventsKey) (Decision, error) {
	var primary Decision
	primarySet := false

	if l == nil || l.client == nil {
		return Decision{Allowed: true, LimitDisabled: true, Scope: "none"}, nil
	}

	if subID := strings.TrimSpace(key.SubscriptionID); subID != "" && l.subLimit > 0 {
		res, err := l.allowKey(ctx, fmt.Sprintf("rl:usage_events:sub:%s:%s", key.OrgID, subID), l.subLimit)
		if err != nil {
			return Decision{}, err
		}
		res.Scope = "subscription"
		if !res.Allowed {
			res.Reason = "subscription_rate_limit"
		}
		primary = res
		primarySet = true
		if !res.Allowed {
			return res, nil
		}
	}

	if custID := strings.TrimSpace(key.CustomerID); custID != "" && l.customerLimit > 0 {
		res, err := l.allowKey(ctx, fmt.Sprintf("rl:usage_events:cus:%s:%s", key.OrgID, custID), l.customerLimit)
		if err != nil {
			return Decision{}, err
		}
		res.Scope = "customer"
		if !res.Allowed {
			res.Reason = "customer_rate_limit"
		}
		if !primarySet {
			primary = res
			primarySet = true
		}
		if !res.Allowed {
			return res, nil
		}
	}

	if orgID := strings.TrimSpace(key.OrgID); orgID != "" && l.orgLimit > 0 {
		res, err := l.allowKey(ctx, fmt.Sprintf("rl:usage_events:org:%s", orgID), l.orgLimit)
		if err != nil {
			return Decision{}, err
		}
		res.Scope = "org"
		if !res.Allowed {
			res.Reason = "org_rate_limit"
		}
		if !primarySet {
			primary = res
			primarySet = true
		}
		if !res.Allowed {
			return res, nil
		}
	}

	if primarySet {
		return primary, nil
	}
	return Decision{Allowed: true, LimitDisabled: true, Scope: "none"}, nil
}

func (l *Limiter) allowKey(ctx context.Context, key string, limit int) (Decision, error) {
	if limit <= 0 {
		return Decision{Allowed: true, LimitDisabled: true}, nil
	}
	count, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return Decision{}, err
	}
	if count == 1 {
		_, _ = l.client.Expire(ctx, key, l.window).Result()
	}
	ttl, ttlErr := l.client.TTL(ctx, key).Result()
	reset := int(l.window.Seconds())
	if ttlErr == nil && ttl > 0 {
		reset = int(ttl.Seconds())
	}
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:      count <= int64(limit),
		Limit:        limit,
		Remaining:    remaining,
		ResetSeconds: reset,
	}, nil
}
