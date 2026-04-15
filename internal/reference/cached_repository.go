package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/railzwaylabs/railzway/internal/reference/domain"
	"github.com/redis/go-redis/v9"
)

const referenceCacheTTL = 15 * time.Second

type cachedRepository struct {
	base  domain.Repository
	cache *redis.Client
}

func NewCachedRepository(base domain.Repository, cache *redis.Client) domain.Repository {
	if cache == nil {
		return base
	}
	return &cachedRepository{base: base, cache: cache}
}

func (r *cachedRepository) ListCountries(ctx context.Context) ([]domain.Country, error) {
	const key = "cache:reference:countries"
	if cached, ok := r.getCached(ctx, key); ok {
		var countries []domain.Country
		if err := json.Unmarshal(cached, &countries); err == nil {
			return countries, nil
		}
	}
	countries, err := r.base.ListCountries(ctx)
	if err != nil {
		return nil, err
	}
	r.setCached(ctx, key, countries)
	return countries, nil
}

func (r *cachedRepository) ListTimezonesByCountry(ctx context.Context, countryCode string) ([]domain.Timezone, error) {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	key := fmt.Sprintf("cache:reference:timezones:%s", code)
	if cached, ok := r.getCached(ctx, key); ok {
		var timezones []domain.Timezone
		if err := json.Unmarshal(cached, &timezones); err == nil {
			return timezones, nil
		}
	}
	timezones, err := r.base.ListTimezonesByCountry(ctx, code)
	if err != nil {
		return nil, err
	}
	r.setCached(ctx, key, timezones)
	return timezones, nil
}

func (r *cachedRepository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	const key = "cache:reference:currencies"
	if cached, ok := r.getCached(ctx, key); ok {
		var currencies []domain.Currency
		if err := json.Unmarshal(cached, &currencies); err == nil {
			return currencies, nil
		}
	}
	currencies, err := r.base.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	r.setCached(ctx, key, currencies)
	return currencies, nil
}

func (r *cachedRepository) getCached(ctx context.Context, key string) ([]byte, bool) {
	if r.cache == nil {
		return nil, false
	}
	raw, err := r.cache.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	return []byte(raw), true
}

func (r *cachedRepository) setCached(ctx context.Context, key string, value interface{}) {
	if r.cache == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = r.cache.Set(ctx, key, raw, referenceCacheTTL).Err()
}
