package config

import (
	"os"
	"strings"
)

const (
	envPrefixLocal       = "AUTH_LOCAL_"
	envPrefixOAuth       = "AUTH_OAUTH_"
	envPrefixGitHub      = "AUTH_GITHUB_"
	envPrefixGoogle      = "AUTH_GOOGLE_"
	envPrefixRailzwayCom = "AUTH_RAILZWAY_COM_"
)

type providerEnvSpec struct {
	providerType string
	prefix       string
	displayName  string
}

var providerSpecs = []providerEnvSpec{
	{providerType: "local", prefix: envPrefixLocal, displayName: "Local"},
	{providerType: "oauth", prefix: envPrefixOAuth, displayName: "OAuth"},
	{providerType: "github", prefix: envPrefixGitHub, displayName: "GitHub"},
	{providerType: "google", prefix: envPrefixGoogle, displayName: "Google"},
	{providerType: "railzway_com", prefix: envPrefixRailzwayCom, displayName: "Railzway.com"},
}

// ParseAuthProvidersFromEnv reads auth provider configuration from environment variables.
func ParseAuthProvidersFromEnv() map[string]AuthProviderConfig {
	env := os.Environ()
	configs := make(map[string]AuthProviderConfig, len(providerSpecs))
	for _, spec := range providerSpecs {
		if !hasEnvPrefix(env, spec.prefix) {
			continue
		}
		cfg := parseProviderConfig(spec.providerType, spec.prefix, spec.displayName)
		configs[cfg.Type] = cfg
	}
	return configs
}

func parseProviderConfig(providerType string, prefix string, defaultName string) AuthProviderConfig {
	name := strings.TrimSpace(getenv(prefix + "NAME"))
	if name == "" {
		if strings.TrimSpace(defaultName) != "" {
			name = defaultName
		} else {
			name = providerType
		}
	}
	return AuthProviderConfig{
		Name:         name,
		Type:         providerType,
		Enabled:      getenvBool(prefix+"ENABLED", false),
		ClientID:     strings.TrimSpace(getenv(prefix + "CLIENT_ID")),
		ClientSecret: strings.TrimSpace(getenv(prefix + "CLIENT_SECRET")),
		AuthURL:      strings.TrimSpace(getenv(prefix + "AUTH_URL")),
		TokenURL:     strings.TrimSpace(getenv(prefix + "TOKEN_URL")),
		APIURL:       strings.TrimSpace(getenv(prefix + "API_URL")),
		Scopes:       parseScopes(getenv(prefix + "SCOPES")),
		AllowSignUp:  getenvBoolFirst([]string{prefix + "ALLOW_SIGNUP", prefix + "ALLOW_SIGN_UP"}, false),
	}
}

func getenv(key string) string {
	return os.Getenv(key)
}

func getenvBool(key string, def bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return parseBool(value, def)
}

func getenvBoolFirst(keys []string, def bool) bool {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return parseBool(value, def)
		}
	}
	return def
}

func parseBool(value string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func parseScopes(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func hasEnvPrefix(env []string, prefix string) bool {
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}
