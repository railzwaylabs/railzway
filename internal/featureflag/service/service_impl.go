package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/featureflag/domain"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

const (
	defaultRollout = 0
	defaultEnabled = false
	cacheTTL       = 60 * time.Second
)

type Service struct {
	repo  domain.Repository
	cache *redis.Client

	mu       sync.RWMutex
	defaults map[string]domain.FeatureFlag
}

func NewService(repo domain.Repository, cache *redis.Client) *Service {
	s := &Service{
		repo:     repo,
		cache:    cache,
		defaults: map[string]domain.FeatureFlag{},
	}
	s.loadDefaults()
	return s
}

type fileFlag struct {
	Key     string `mapstructure:"key"`
	Enabled bool   `mapstructure:"enabled"`
	Rollout int    `mapstructure:"rollout"`
}

type fileConfig struct {
	FeatureFlags []fileFlag `mapstructure:"feature_flags"`
}

func (s *Service) loadDefaults() {
	v := viper.New()
	v.AddConfigPath("/var/lib/railzway/config")
	v.AddConfigPath(".")
	v.SetConfigName("features")
	v.SetConfigType("yml")

	if err := v.ReadInConfig(); err != nil {
		log.Printf("feature flags: no config loaded: %v", err)
	}

	reload := func() {
		var cfg fileConfig
		if err := v.Unmarshal(&cfg); err != nil {
			log.Printf("feature flags: failed to unmarshal: %v", err)
			return
		}

		defaults := map[string]domain.FeatureFlag{}
		for _, item := range cfg.FeatureFlags {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				continue
			}

			rollout := item.Rollout
			if rollout < 0 {
				rollout = 0
			}

			if rollout > 100 {
				rollout = 100
			}

			defaults[key] = domain.FeatureFlag{
				Key:     key,
				Enabled: item.Enabled,
				Rollout: rollout,
			}
		}

		s.mu.Lock()
		s.defaults = defaults
		s.mu.Unlock()
		log.Printf("feature flags: defaults reloaded (%d flags)", len(defaults))
	}

	reload()
	v.WatchConfig()
	v.OnConfigChange(func(_ fsnotify.Event) { reload() })
}

func (s *Service) IsEnabled(ctx context.Context, orgID, key string) bool {
	k := strings.TrimSpace(key)
	if k == "" {
		return false
	}

	org := strings.TrimSpace(orgID)
	if org != "" {
		if cached, ok := s.getCached(ctx, org, k); ok {
			return evalFlag(cached, org, k)
		}

		orgUUID, err := uuid.Parse(org)
		if err == nil && orgUUID != uuid.Nil {
			flag, err := s.repo.FindFlag(ctx, &orgUUID, k)
			if err == nil && flag != nil {
				_ = s.setCache(ctx, org, k, *flag)
				return evalFlag(*flag, org, k)
			}
		}
	}

	if cached, ok := s.getCached(ctx, "", k); ok {
		return evalFlag(cached, org, k)
	}

	flag, err := s.repo.FindFlag(ctx, nil, k)
	if err == nil && flag != nil {
		_ = s.setCache(ctx, "", k, *flag)
		return evalFlag(*flag, org, k)
	}

	s.mu.RLock()
	defaultFlag, ok := s.defaults[k]
	s.mu.RUnlock()
	if ok {
		return evalFlag(defaultFlag, org, k)
	}

	return evalFlag(domain.FeatureFlag{Enabled: defaultEnabled, Rollout: defaultRollout}, org, k)
}

type FlagView struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Rollout int    `json:"rollout"`
	Source  string `json:"source"`
}

func (s *Service) ListEffectiveFlags(ctx context.Context, orgID string) ([]FlagView, error) {
	flags := map[string]FlagView{}

	s.mu.RLock()
	for key, flag := range s.defaults {
		flags[key] = FlagView{
			Key:     key,
			Enabled: flag.Enabled,
			Rollout: flag.Rollout,
			Source:  "default",
		}
	}
	s.mu.RUnlock()

	globalFlags, err := s.repo.ListFlags(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, flag := range globalFlags {
		flags[flag.Key] = FlagView{
			Key:     flag.Key,
			Enabled: flag.Enabled,
			Rollout: flag.Rollout,
			Source:  "global",
		}
	}

	org := strings.TrimSpace(orgID)
	if org != "" {
		orgUUID, err := uuid.Parse(org)
		if err != nil || orgUUID == uuid.Nil {
			return nil, fmt.Errorf("invalid_org")
		}

		tenantFlags, err := s.repo.ListFlags(ctx, &orgUUID)
		if err != nil {
			return nil, err
		}

		for _, flag := range tenantFlags {
			flags[flag.Key] = FlagView{
				Key:     flag.Key,
				Enabled: flag.Enabled,
				Rollout: flag.Rollout,
				Source:  "tenant",
			}
		}
	}

	keys := make([]string, 0, len(flags))
	for key := range flags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]FlagView, 0, len(keys))
	for _, key := range keys {
		result = append(result, flags[key])
	}

	return result, nil
}

func (s *Service) UpsertFlag(ctx context.Context, actorID string, orgID *string, key string, enabled bool, rollout int) error {
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		_, actorUUID := auditlog.ActorFromContext(ctx)
		if actorUUID != nil {
			actor = actorUUID.String()
		}
	}

	if actor == "" {
		return fmt.Errorf("invalid_actor")
	}

	k := strings.TrimSpace(key)
	if k == "" {
		return fmt.Errorf("invalid_key")
	}

	if rollout < 0 || rollout > 100 {
		return fmt.Errorf("invalid_rollout")
	}

	var orgUUID *uuid.UUID
	if orgID != nil {
		val := strings.TrimSpace(*orgID)
		if val != "" {
			parsed, err := uuid.Parse(val)
			if err != nil || parsed == uuid.Nil {
				return fmt.Errorf("invalid_org")
			}
			orgUUID = &parsed
		}
	}

	now := time.Now().UTC()
	flag := domain.FeatureFlag{
		ID:        uuid.New(),
		OrgID:     orgUUID,
		Key:       k,
		Enabled:   enabled,
		Rollout:   rollout,
		CreatedAt: now,
		UpdatedAt: now,
	}

	audit := domain.FeatureFlagAudit{
		ID:        uuid.New(),
		OrgID:     orgUUID,
		Key:       k,
		Enabled:   enabled,
		Rollout:   rollout,
		ActorID:   actor,
		CreatedAt: now,
	}

	if err := s.repo.UpsertFlag(ctx, flag); err != nil {
		return err
	}

	if err := s.repo.CreateAudit(ctx, audit); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.Del(ctx, cacheKey(orgUUID, k)).Err()
	}

	return nil
}

func evalFlag(flag domain.FeatureFlag, orgID string, key string) bool {
	if !flag.Enabled {
		return false
	}

	if flag.Rollout <= 0 {
		return false
	}

	if flag.Rollout >= 100 {
		return true
	}

	seed := key
	if orgID != "" {
		seed = orgID + ":" + key
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	percent := int(h.Sum32() % 100)
	return percent < flag.Rollout
}

func (s *Service) getCached(ctx context.Context, orgID, key string) (domain.FeatureFlag, bool) {
	if s.cache == nil {
		return domain.FeatureFlag{}, false
	}
	raw, err := s.cache.Get(ctx, cacheKeyString(orgID, key)).Result()
	if err != nil || raw == "" {
		return domain.FeatureFlag{}, false
	}
	var flag domain.FeatureFlag
	if err := json.Unmarshal([]byte(raw), &flag); err != nil {
		return domain.FeatureFlag{}, false
	}
	return flag, true
}

func (s *Service) setCache(ctx context.Context, orgID, key string, flag domain.FeatureFlag) error {
	if s.cache == nil {
		return nil
	}
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, cacheKeyString(orgID, key), data, cacheTTL).Err()
}

func cacheKey(orgID *uuid.UUID, key string) string {
	if orgID == nil {
		return cacheKeyString("", key)
	}
	return cacheKeyString(orgID.String(), key)
}

func cacheKeyString(orgID, key string) string {
	if strings.TrimSpace(orgID) == "" {
		return fmt.Sprintf("ff:global:%s", key)
	}
	return fmt.Sprintf("ff:%s:%s", orgID, key)
}
