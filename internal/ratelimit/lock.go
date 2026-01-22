package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const lockReleaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

type Locker struct {
	client *redis.Client
	script *redis.Script
}

func NewLocker(client *redis.Client) *Locker {
	if client == nil {
		return nil
	}
	return &Locker{
		client: client,
		script: redis.NewScript(lockReleaseScript),
	}
}

func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if l == nil || l.client == nil {
		return "", false, errors.New("lock client not configured")
	}
	if key == "" {
		return "", false, errors.New("lock key is empty")
	}
	if ttl <= 0 {
		return "", false, errors.New("lock ttl must be positive")
	}

	token := uuid.NewString()
	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	return token, ok, nil
}

func (l *Locker) Release(ctx context.Context, key, token string) error {
	if l == nil || l.client == nil {
		return nil
	}
	if key == "" || token == "" {
		return nil
	}
	return l.script.Run(ctx, l.client, []string{key}, token).Err()
}
