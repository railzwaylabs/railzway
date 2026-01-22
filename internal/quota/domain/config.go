package domain

import (
	"os"
	"strconv"
)

type Config struct {
	Enabled bool

	// Org-level quotas
	OrgCustomer     int // Max customers per org
	OrgSubscription int // Max subscriptions per org
	OrgUsageMonthly int // Max usage events per org per month
}

func LoadFromEnv() *Config {
	return &Config{
		Enabled:         getEnvBool("QUOTA_ENABLED", true),
		OrgCustomer:     getEnvInt("QUOTA_ORG_CUSTOMER", 100),
		OrgSubscription: getEnvInt("QUOTA_ORG_SUBSCRIPTION", 10),
		OrgUsageMonthly: getEnvInt("QUOTA_ORG_USAGE_MONTHLY", 100000),
	}
}

func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
