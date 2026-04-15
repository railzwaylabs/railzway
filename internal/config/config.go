package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	AppConfig
	DBConfig
	RedisConfig
	CacheConfig
	SummaryConfig
	SubscriptionConfig
	RatingConfig
	ReconciliationConfig
	RateLimitConfig
	AppsConfig
	StripeConfig
	PublicLinkConfig
	BootstrapConfig
	SessionConfig
}

type AppEnv string

const (
	AppEnvDevelopment AppEnv = "development"
	AppEnvProduction  AppEnv = "production"
)

func (e AppEnv) IsProduction() bool {
	return e == AppEnvProduction
}

func (e AppEnv) IsDevelopment() bool {
	return e == AppEnvDevelopment
}

type TLSMode string

const (
	TLSModeDisabled TLSMode = "disabled"
	TLSModeDirect   TLSMode = "direct"
	TLSModeProxy    TLSMode = "proxy"
)

func (m TLSMode) IsDisabled() bool {
	return m == "" || m == TLSModeDisabled
}

func (m TLSMode) IsDirect() bool {
	return m == TLSModeDirect
}

func (m TLSMode) IsProxy() bool {
	return m == TLSModeProxy
}

type AppConfig struct {
	AppName            string  `mapstructure:"APP_NAME"`
	AppEnv             AppEnv  `mapstructure:"APP_ENV"`
	AppPort            int     `mapstructure:"PORT"`
	AppTLSMode         TLSMode `mapstructure:"APP_TLS_MODE"`
	AppTLSCertFile     string  `mapstructure:"APP_TLS_CERT_FILE"`
	AppTLSKeyFile      string  `mapstructure:"APP_TLS_KEY_FILE"`
	CSPExtraDirectives string  `mapstructure:"CSP_EXTRA_DIRECTIVES"`
}

type DBConfig struct {
	DBType     string `mapstructure:"DB_TYPE"`
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPass     string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`
	DBTimezone string `mapstructure:"DB_TIMEZONE"`

	DBMaxIdleConn     int `mapstructure:"DB_MAX_IDLE_CONN"`
	DBMaxOpenConn     int `mapstructure:"DB_MAX_OPEN_CONN"`
	DBConnMaxLifetime int `mapstructure:"DB_CONN_MAX_LIFETIME"`
	DBConnMaxIdleTime int `mapstructure:"DB_CONN_MAX_IDLE_TIME"`
}

type RedisConfig struct {
	RedisURL      string `mapstructure:"REDIS_URL"`
	RedisUsername string `mapstructure:"REDIS_USERNAME"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`
}

type CacheConfig struct {
	CacheURL      string `mapstructure:"CACHE_URL"`
	CacheUsername string `mapstructure:"CACHE_USERNAME"`
	CachePassword string `mapstructure:"CACHE_PASSWORD"`
	CacheDB       int    `mapstructure:"CACHE_DB"`
}

type SummaryConfig struct {
	SummaryConcurrency        int `mapstructure:"SUMMARY_CONCURRENCY"`
	SummaryTimeoutMs          int `mapstructure:"SUMMARY_TIMEOUT_MS"`
	SummaryRefreshIntervalSec int `mapstructure:"SUMMARY_REFRESH_INTERVAL_SEC"`
}

type SubscriptionConfig struct {
	SubscriptionClosePeriodIntervalSec int `mapstructure:"SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC"`
	SubscriptionClosePeriodBatchSize   int `mapstructure:"SUBSCRIPTION_CLOSE_PERIOD_BATCH_SIZE"`
}

type RatingConfig struct {
	RatingJobIntervalSec int `mapstructure:"RATING_JOB_INTERVAL_SEC"`
	RatingJobBatchSize   int `mapstructure:"RATING_JOB_BATCH_SIZE"`
	LateUsageGraceHours  int `mapstructure:"LATE_USAGE_GRACE_HOURS"`
}

type ReconciliationConfig struct {
	ReconciliationJobIntervalSec int `mapstructure:"RECONCILIATION_JOB_INTERVAL_SEC"`
	ReconciliationWindowDays     int `mapstructure:"RECONCILIATION_WINDOW_DAYS"`
	ReconciliationInvoiceLimit   int `mapstructure:"RECONCILIATION_INVOICE_LIMIT"`
}

type RateLimitConfig struct {
	UsageEventsWindowSec                   int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC"`
	UsageEventsSubscriptionPerMin          int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN"`
	UsageEventsCustomerPerMin              int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN"`
	UsageEventsOrgPerMin                   int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN"`
	UsageEventsConcurrencyPerCustomerMeter int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER"`
	UsageEventsConcurrencyTTLSeconds       int `mapstructure:"RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC"`
}

type AppsConfig struct {
	AppsCredentialsKey string `mapstructure:"APPS_CREDENTIALS_KEY"`
}

type StripeConfig struct {
	StripeConnectClientID    string `mapstructure:"STRIPE_CONNECT_CLIENT_ID"`
	StripeConnectSecret      string `mapstructure:"STRIPE_CONNECT_SECRET"`
	StripeConnectRedirectURL string `mapstructure:"STRIPE_CONNECT_REDIRECT_URL"`
}

type PublicLinkConfig struct {
	PublicLinkSecret   string `mapstructure:"PUBLIC_LINK_SECRET"`
	PublicLinkTTLHours int    `mapstructure:"PUBLIC_LINK_TTL_HOURS"`
	PublicLinkBaseURL  string `mapstructure:"PUBLIC_LINK_BASE_URL"`
}

type BootstrapConfig struct {
	EnsureDefaultOrgAndUser bool   `mapstructure:"ENSURE_DEFAULT_ORG_AND_USER"`
	OrgName                 string `mapstructure:"RAILZWAY_ORG_NAME"`
	UserEmail               string `mapstructure:"RAILZWAY_USER_EMAIL"`
	UserPassword            string `mapstructure:"RAILZWAY_USER_PASSWORD"`
}

type SessionConfig struct {
	SessionTTLHours int    `mapstructure:"SESSION_TTL_HOURS"`
	SessionSecret   string `mapstructure:"SESSION_SECRET"`
	SessionCookie   string `mapstructure:"SESSION_COOKIE_NAME"`
}

func Register() (*Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.SetDefault("SESSION_TTL_HOURS", 24)
	v.SetDefault("SESSION_COOKIE_NAME", "rz_admin_session")
	v.SetDefault("PUBLIC_LINK_TTL_HOURS", 168)
	v.SetDefault("LATE_USAGE_GRACE_HOURS", 0)
	v.SetDefault("RECONCILIATION_JOB_INTERVAL_SEC", 21600)
	v.SetDefault("RECONCILIATION_WINDOW_DAYS", 7)
	v.SetDefault("RECONCILIATION_INVOICE_LIMIT", 200)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC", 60)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN", 120)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN", 600)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN", 3000)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER", 1)
	v.SetDefault("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC", 5)

	v.SetEnvKeyReplacer(
		strings.NewReplacer(".", "_"),
	)

	v.AutomaticEnv()
	bindEnvKeys(v)

	v.AddConfigPath("/var/lib/railzway/config")
	v.AddConfigPath(".")

	v.SetConfigName("config")
	v.SetConfigType("yml")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			zap.L().Fatal("failed to read config", zap.Error(err))
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(&cfg)
	// 🔥 HOT RELOAD
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		zap.L().Info("config changed", zap.String("file", e.Name))
	})

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {

	// App
	applyEnvString("APP_NAME", func(v string) { cfg.AppName = v })
	applyEnvString("APP_ENV", func(v string) { cfg.AppEnv = AppEnv(v) })
	applyEnvInt("PORT", func(v int) { cfg.AppPort = v })
	applyEnvString("APP_TLS_MODE", func(v string) { cfg.AppTLSMode = TLSMode(v) })
	applyEnvString("APP_TLS_CERT_FILE", func(v string) { cfg.AppTLSCertFile = v })
	applyEnvString("APP_TLS_KEY_FILE", func(v string) { cfg.AppTLSKeyFile = v })
	applyEnvString("CSP_EXTRA_DIRECTIVES", func(v string) { cfg.CSPExtraDirectives = v })

	// Database
	applyEnvString("DB_TYPE", func(v string) { cfg.DBType = v })
	applyEnvString("DB_HOST", func(v string) { cfg.DBHost = v })
	applyEnvString("DB_PORT", func(v string) { cfg.DBPort = v })
	applyEnvString("DB_USER", func(v string) { cfg.DBUser = v })
	applyEnvString("DB_PASSWORD", func(v string) { cfg.DBPass = v })
	applyEnvString("DB_NAME", func(v string) { cfg.DBName = v })
	applyEnvString("DB_SSL_MODE", func(v string) { cfg.DBSSLMode = v })
	applyEnvString("DB_TIMEZONE", func(v string) { cfg.DBTimezone = v })
	applyEnvInt("DB_MAX_IDLE_CONN", func(v int) { cfg.DBMaxIdleConn = v })
	applyEnvInt("DB_MAX_OPEN_CONN", func(v int) { cfg.DBMaxOpenConn = v })
	applyEnvInt("DB_CONN_MAX_LIFETIME", func(v int) { cfg.DBConnMaxLifetime = v })
	applyEnvInt("DB_CONN_MAX_IDLE_TIME", func(v int) { cfg.DBConnMaxIdleTime = v })

	// Redis (sessions/idempotency)
	applyEnvString("REDIS_URL", func(v string) { cfg.RedisURL = v })
	applyEnvString("REDIS_USERNAME", func(v string) { cfg.RedisUsername = v })
	applyEnvString("REDIS_PASSWORD", func(v string) { cfg.RedisPassword = v })
	applyEnvInt("REDIS_DB", func(v int) { cfg.RedisDB = v })

	// Cache
	applyEnvString("CACHE_URL", func(v string) { cfg.CacheURL = v })
	applyEnvString("CACHE_USERNAME", func(v string) { cfg.CacheUsername = v })
	applyEnvString("CACHE_PASSWORD", func(v string) { cfg.CachePassword = v })
	applyEnvInt("CACHE_DB", func(v int) { cfg.CacheDB = v })

	// Summaries
	applyEnvInt("SUMMARY_CONCURRENCY", func(v int) { cfg.SummaryConcurrency = v })
	applyEnvInt("SUMMARY_TIMEOUT_MS", func(v int) { cfg.SummaryTimeoutMs = v })
	applyEnvInt("SUMMARY_REFRESH_INTERVAL_SEC", func(v int) { cfg.SummaryRefreshIntervalSec = v })
	// Subscription scheduling
	applyEnvInt("SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC", func(v int) { cfg.SubscriptionClosePeriodIntervalSec = v })
	applyEnvInt("SUBSCRIPTION_CLOSE_PERIOD_BATCH_SIZE", func(v int) { cfg.SubscriptionClosePeriodBatchSize = v })

	// Rating scheduling
	applyEnvInt("RATING_JOB_INTERVAL_SEC", func(v int) { cfg.RatingJobIntervalSec = v })
	applyEnvInt("RATING_JOB_BATCH_SIZE", func(v int) { cfg.RatingJobBatchSize = v })

	// Public API rate limits
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC", func(v int) { cfg.RateLimitConfig.UsageEventsWindowSec = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN", func(v int) { cfg.RateLimitConfig.UsageEventsSubscriptionPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN", func(v int) { cfg.RateLimitConfig.UsageEventsCustomerPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN", func(v int) { cfg.RateLimitConfig.UsageEventsOrgPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER", func(v int) {
		cfg.RateLimitConfig.UsageEventsConcurrencyPerCustomerMeter = v
	})
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC", func(v int) {
		cfg.RateLimitConfig.UsageEventsConcurrencyTTLSeconds = v
	})

	// Apps & providers
	applyEnvString("APPS_CREDENTIALS_KEY", func(v string) { cfg.AppsCredentialsKey = v })
	applyEnvString("STRIPE_CONNECT_CLIENT_ID", func(v string) { cfg.StripeConnectClientID = v })
	applyEnvString("STRIPE_CONNECT_SECRET", func(v string) { cfg.StripeConnectSecret = v })
	applyEnvString("STRIPE_CONNECT_REDIRECT_URL", func(v string) { cfg.StripeConnectRedirectURL = v })

	// Public links
	applyEnvString("PUBLIC_LINK_SECRET", func(v string) { cfg.PublicLinkSecret = v })
	applyEnvInt("PUBLIC_LINK_TTL_HOURS", func(v int) { cfg.PublicLinkTTLHours = v })
	applyEnvString("PUBLIC_LINK_BASE_URL", func(v string) { cfg.PublicLinkBaseURL = v })

	// Bootstrap
	applyEnvBool("ENSURE_DEFAULT_ORG_AND_USER", func(v bool) { cfg.BootstrapConfig.EnsureDefaultOrgAndUser = v })
	applyEnvString("RAILZWAY_ORG_NAME", func(v string) { cfg.BootstrapConfig.OrgName = v })
	applyEnvString("RAILZWAY_USER_EMAIL", func(v string) { cfg.BootstrapConfig.UserEmail = v })
	applyEnvString("RAILZWAY_USER_PASSWORD", func(v string) { cfg.BootstrapConfig.UserPassword = v })

	// Sessions
	applyEnvInt("SESSION_TTL_HOURS", func(v int) { cfg.SessionConfig.SessionTTLHours = v })
	applyEnvString("SESSION_SECRET", func(v string) { cfg.SessionConfig.SessionSecret = v })
	applyEnvString("SESSION_COOKIE_NAME", func(v string) { cfg.SessionConfig.SessionCookie = v })
}

func applyEnvString(key string, apply func(string)) {
	if v, ok := envString(key); ok {
		apply(v)
	}
}

func applyEnvInt(key string, apply func(int)) {
	if v, ok := envInt(key); ok {
		apply(v)
	}
}

func applyEnvBool(key string, apply func(bool)) {
	if v, ok := envBool(key); ok {
		apply(v)
	}
}

func envString(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", false
	}
	return val, true
}

func envInt(key string) (int, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return 0, false
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func envBool(key string) (bool, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return false, false
	}
	val = strings.TrimSpace(strings.ToLower(val))
	if val == "" {
		return false, false
	}
	switch val {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func bindEnvKeys(v *viper.Viper) {
	keys := []string{
		"APP_NAME",
		"APP_ENV",
		"PORT",
		"APP_TLS_MODE",
		"APP_TLS_CERT_FILE",
		"APP_TLS_KEY_FILE",
		"CSP_EXTRA_DIRECTIVES",
		"DB_TYPE",
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_SSL_MODE",
		"DB_TIMEZONE",
		"DB_MAX_IDLE_CONN",
		"DB_MAX_OPEN_CONN",
		"DB_CONN_MAX_LIFETIME",
		"DB_CONN_MAX_IDLE_TIME",
		"REDIS_URL",
		"REDIS_USERNAME",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"CACHE_URL",
		"CACHE_USERNAME",
		"CACHE_PASSWORD",
		"CACHE_DB",
		"SUMMARY_CONCURRENCY",
		"SUMMARY_TIMEOUT_MS",
		"SUMMARY_REFRESH_INTERVAL_SEC",
		"SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC",
		"SUBSCRIPTION_CLOSE_PERIOD_BATCH_SIZE",
		"RATING_JOB_INTERVAL_SEC",
		"RATING_JOB_BATCH_SIZE",
		"RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC",
		"RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN",
		"RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN",
		"RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN",
		"RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER",
		"RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC",
		"APPS_CREDENTIALS_KEY",
		"STRIPE_CONNECT_CLIENT_ID",
		"STRIPE_CONNECT_SECRET",
		"STRIPE_CONNECT_REDIRECT_URL",
		"PUBLIC_LINK_SECRET",
		"PUBLIC_LINK_TTL_HOURS",
		"PUBLIC_LINK_BASE_URL",
		"ENSURE_DEFAULT_ORG_AND_USER",
		"RAILZWAY_ORG_NAME",
		"RAILZWAY_USER_EMAIL",
		"RAILZWAY_USER_PASSWORD",
		"SESSION_TTL_HOURS",
		"SESSION_SECRET",
		"SESSION_COOKIE_NAME",
	}

	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}
