package config

import (
	"log"
	"sort"
	"strings"

	"github.com/smallbiznis/railzway/internal/auth/features"
)

// AuthProviderRegistry captures parsed providers and activation state.
type AuthProviderRegistry struct {
	All     map[string]AuthProviderConfig
	Active  map[string]AuthProviderConfig
	Ignored map[string]string
}

// BuildAuthProviderRegistry builds a registry from parsed provider configs.
func BuildAuthProviderRegistry(cfgs map[string]AuthProviderConfig) AuthProviderRegistry {
	registry := AuthProviderRegistry{
		All:     make(map[string]AuthProviderConfig, len(cfgs)),
		Active:  make(map[string]AuthProviderConfig),
		Ignored: make(map[string]string),
	}

	for key, cfg := range cfgs {
		cfg = normalizeProviderConfig(key, cfg)
		registry.All[cfg.Type] = cfg
	}

	keys := make([]string, 0, len(registry.All))
	for key := range registry.All {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		cfg := registry.All[key]
		if !cfg.Enabled {
			log.Printf("[auth] provider=%s enabled=false -> DISABLED", cfg.Type)
			continue
		}
		if !features.ImplementedAuthFeatures[cfg.Type] {
			registry.Ignored[cfg.Type] = "enabled in config but feature not implemented"
			log.Printf("[auth] provider=%s enabled=true implemented=false -> IGNORED", cfg.Type)
			continue
		}
		registry.Active[cfg.Type] = cfg
		log.Printf("[auth] provider=%s enabled=true implemented=true -> ACTIVE", cfg.Type)
	}

	return registry
}

func normalizeProviderConfig(key string, cfg AuthProviderConfig) AuthProviderConfig {
	if cfg.Type == "" {
		cfg.Type = key
	}
	cfg.Type = normalizeProviderType(cfg.Type)
	if cfg.Name == "" {
		cfg.Name = cfg.Type
	}
	return cfg
}

func normalizeProviderType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "railzway_com":
		return "railzway_com"
	default:
		return value
	}
}
