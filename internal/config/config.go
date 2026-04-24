package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	DB           DBConfig           `mapstructure:"db"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Cache        CacheConfig        `mapstructure:"cache"`
	Billing      BillingConfig      `mapstructure:"billing"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limit"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
	PublicLink   PublicLinkConfig   `mapstructure:"public_link"`
	Bootstrap    BootstrapConfig    `mapstructure:"bootstrap"`
	Session      SessionConfig      `mapstructure:"session"`
	AIWorkflow   AIWorkflowConfig   `mapstructure:"ai_workflow"`
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
	Name               string  `mapstructure:"name"`
	Env                AppEnv  `mapstructure:"env"`
	Port               int     `mapstructure:"port"`
	TLSMode            TLSMode `mapstructure:"tls_mode"`
	TLSCertFile        string  `mapstructure:"tls_cert_file"`
	TLSKeyFile         string  `mapstructure:"tls_key_file"`
	CSPExtraDirectives string  `mapstructure:"csp_extra_directives"`
}

type DBConfig struct {
	Type            string `mapstructure:"type"`
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Pass            string `mapstructure:"password"`
	Name            string `mapstructure:"name"`
	SSLMode         string `mapstructure:"ssl_mode"`
	Timezone        string `mapstructure:"timezone"`
	MaxIdleConn     int    `mapstructure:"max_idle_conn"`
	MaxOpenConn     int    `mapstructure:"max_open_conn"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type CacheConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type BillingConfig struct {
	SummaryConcurrency                 int `mapstructure:"summary_concurrency"`
	SummaryTimeoutMs                   int `mapstructure:"summary_timeout_ms"`
	SummaryRefreshIntervalSec          int `mapstructure:"summary_refresh_interval_sec"`
	SubscriptionClosePeriodIntervalSec int `mapstructure:"subscription_close_period_interval_sec"`
	SubscriptionClosePeriodBatchSize   int `mapstructure:"subscription_close_period_batch_size"`
	RatingJobIntervalSec               int `mapstructure:"rating_job_interval_sec"`
	RatingJobBatchSize                 int `mapstructure:"rating_job_batch_size"`
	LateUsageGraceHours                int `mapstructure:"late_usage_grace_hours"`
	ReconciliationJobIntervalSec       int `mapstructure:"reconciliation_job_interval_sec"`
	ReconciliationWindowDays           int `mapstructure:"reconciliation_window_days"`
	ReconciliationInvoiceLimit         int `mapstructure:"reconciliation_invoice_limit"`
}

type RateLimitConfig struct {
	UsageEventsWindowSec                   int `mapstructure:"usage_events_window_sec"`
	UsageEventsSubscriptionPerMin          int `mapstructure:"usage_events_subscription_per_min"`
	UsageEventsCustomerPerMin              int `mapstructure:"usage_events_customer_per_min"`
	UsageEventsOrgPerMin                   int `mapstructure:"usage_events_org_per_min"`
	UsageEventsConcurrencyPerCustomerMeter int `mapstructure:"usage_events_concurrency_per_customer_meter"`
	UsageEventsConcurrencyTTLSeconds       int `mapstructure:"usage_events_concurrency_ttl_sec"`
}

type IntegrationsConfig struct {
	AppsCredentialsKey       string `mapstructure:"apps_credentials_key"`
	StripeConnectClientID    string `mapstructure:"stripe_connect_client_id"`
	StripeConnectSecret      string `mapstructure:"stripe_connect_secret"`
	StripeConnectRedirectURL string `mapstructure:"stripe_connect_redirect_url"`
}

type PublicLinkConfig struct {
	Secret   string `mapstructure:"secret"`
	TTLHours int    `mapstructure:"ttl_hours"`
	BaseURL  string `mapstructure:"base_url"`
}

type BootstrapConfig struct {
	EnsureDefaultOrgAndUser bool   `mapstructure:"ensure_default_org_and_user"`
	OrgName                 string `mapstructure:"railzway_org_name"`
	UserEmail               string `mapstructure:"railzway_user_email"`
	UserPassword            string `mapstructure:"railzway_user_password"`
}

type SessionConfig struct {
	TTLHours   int    `mapstructure:"ttl_hours"`
	Secret     string `mapstructure:"secret"`
	CookieName string `mapstructure:"cookie_name"`
}

type AIWorkflowConfig struct {
	GenkitEnabled bool   `mapstructure:"genkit_enabled"`
	Model         string `mapstructure:"model"`
	TimeoutMs     int    `mapstructure:"timeout_ms"`
	APIKey        string `mapstructure:"api_key"`
}

func Register() (*Config, error) {
	_ = godotenv.Load()
	return register("")
}

func RegisterFor(target string) func() (*Config, error) {
	return func() (*Config, error) {
		_ = godotenv.Load()
		return register(target)
	}
}

func register(target string) (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	bindEnvKeys(v)

	v.SetConfigName("base.defaults")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}

	if name := strings.TrimSpace(target); name != "" {
		v.SetConfigName("defaults")
		v.AddConfigPath(
			filepath.Join(".", "config", name),
		)

		if err := v.MergeInConfig(); err != nil {
			// optional: ignore kalau file target tidak ada
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) {
				return nil, fmt.Errorf("failed to merge target config: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
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
	applyEnvString("APP_NAME", func(v string) { cfg.App.Name = v })
	applyEnvString("APP_ENV", func(v string) { cfg.App.Env = AppEnv(v) })
	applyEnvInt("PORT", func(v int) { cfg.App.Port = v })
	applyEnvString("APP_TLS_MODE", func(v string) { cfg.App.TLSMode = TLSMode(v) })
	applyEnvString("APP_TLS_CERT_FILE", func(v string) { cfg.App.TLSCertFile = v })
	applyEnvString("APP_TLS_KEY_FILE", func(v string) { cfg.App.TLSKeyFile = v })
	applyEnvString("CSP_EXTRA_DIRECTIVES", func(v string) { cfg.App.CSPExtraDirectives = v })

	// Database
	applyEnvString("DB_TYPE", func(v string) { cfg.DB.Type = v })
	applyEnvString("DB_HOST", func(v string) { cfg.DB.Host = v })
	applyEnvString("DB_PORT", func(v string) { cfg.DB.Port = v })
	applyEnvString("DB_USER", func(v string) { cfg.DB.User = v })
	applyEnvString("DB_PASSWORD", func(v string) { cfg.DB.Pass = v })
	applyEnvString("DB_NAME", func(v string) { cfg.DB.Name = v })
	applyEnvString("DB_SSL_MODE", func(v string) { cfg.DB.SSLMode = v })
	applyEnvString("DB_TIMEZONE", func(v string) { cfg.DB.Timezone = v })
	applyEnvInt("DB_MAX_IDLE_CONN", func(v int) { cfg.DB.MaxIdleConn = v })
	applyEnvInt("DB_MAX_OPEN_CONN", func(v int) { cfg.DB.MaxOpenConn = v })
	applyEnvInt("DB_CONN_MAX_LIFETIME", func(v int) { cfg.DB.ConnMaxLifetime = v })
	applyEnvInt("DB_CONN_MAX_IDLE_TIME", func(v int) { cfg.DB.ConnMaxIdleTime = v })

	// Redis (sessions/idempotency)
	applyEnvString("REDIS_URL", func(v string) { cfg.Redis.URL = v })
	applyEnvString("REDIS_USERNAME", func(v string) { cfg.Redis.Username = v })
	applyEnvString("REDIS_PASSWORD", func(v string) { cfg.Redis.Password = v })
	applyEnvInt("REDIS_DB", func(v int) { cfg.Redis.DB = v })

	// Cache
	applyEnvString("CACHE_URL", func(v string) { cfg.Cache.URL = v })
	applyEnvString("CACHE_USERNAME", func(v string) { cfg.Cache.Username = v })
	applyEnvString("CACHE_PASSWORD", func(v string) { cfg.Cache.Password = v })
	applyEnvInt("CACHE_DB", func(v int) { cfg.Cache.DB = v })

	// Summaries
	applyEnvInt("SUMMARY_CONCURRENCY", func(v int) { cfg.Billing.SummaryConcurrency = v })
	applyEnvInt("SUMMARY_TIMEOUT_MS", func(v int) { cfg.Billing.SummaryTimeoutMs = v })
	applyEnvInt("SUMMARY_REFRESH_INTERVAL_SEC", func(v int) { cfg.Billing.SummaryRefreshIntervalSec = v })
	// Subscription scheduling
	applyEnvInt("SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC", func(v int) { cfg.Billing.SubscriptionClosePeriodIntervalSec = v })
	applyEnvInt("SUBSCRIPTION_CLOSE_PERIOD_BATCH_SIZE", func(v int) { cfg.Billing.SubscriptionClosePeriodBatchSize = v })

	// Rating scheduling
	applyEnvInt("RATING_JOB_INTERVAL_SEC", func(v int) { cfg.Billing.RatingJobIntervalSec = v })
	applyEnvInt("RATING_JOB_BATCH_SIZE", func(v int) { cfg.Billing.RatingJobBatchSize = v })
	applyEnvInt("LATE_USAGE_GRACE_HOURS", func(v int) { cfg.Billing.LateUsageGraceHours = v })

	// Public API rate limits
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_WINDOW_SEC", func(v int) { cfg.RateLimit.UsageEventsWindowSec = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_SUBSCRIPTION_PER_MIN", func(v int) { cfg.RateLimit.UsageEventsSubscriptionPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CUSTOMER_PER_MIN", func(v int) { cfg.RateLimit.UsageEventsCustomerPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_ORG_PER_MIN", func(v int) { cfg.RateLimit.UsageEventsOrgPerMin = v })
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_PER_CUSTOMER_METER", func(v int) {
		cfg.RateLimit.UsageEventsConcurrencyPerCustomerMeter = v
	})
	applyEnvInt("RATE_LIMIT_USAGE_EVENTS_CONCURRENCY_TTL_SEC", func(v int) {
		cfg.RateLimit.UsageEventsConcurrencyTTLSeconds = v
	})

	// Apps & providers
	applyEnvString("APPS_CREDENTIALS_KEY", func(v string) { cfg.Integrations.AppsCredentialsKey = v })
	applyEnvString("STRIPE_CONNECT_CLIENT_ID", func(v string) { cfg.Integrations.StripeConnectClientID = v })
	applyEnvString("STRIPE_CONNECT_SECRET", func(v string) { cfg.Integrations.StripeConnectSecret = v })
	applyEnvString("STRIPE_CONNECT_REDIRECT_URL", func(v string) { cfg.Integrations.StripeConnectRedirectURL = v })

	// Public links
	applyEnvString("PUBLIC_LINK_SECRET", func(v string) { cfg.PublicLink.Secret = v })
	applyEnvInt("PUBLIC_LINK_TTL_HOURS", func(v int) { cfg.PublicLink.TTLHours = v })
	applyEnvString("PUBLIC_LINK_BASE_URL", func(v string) { cfg.PublicLink.BaseURL = v })

	// Bootstrap
	applyEnvBool("ENSURE_DEFAULT_ORG_AND_USER", func(v bool) { cfg.Bootstrap.EnsureDefaultOrgAndUser = v })
	applyEnvString("RAILZWAY_ORG_NAME", func(v string) { cfg.Bootstrap.OrgName = v })
	applyEnvString("RAILZWAY_USER_EMAIL", func(v string) { cfg.Bootstrap.UserEmail = v })
	applyEnvString("RAILZWAY_USER_PASSWORD", func(v string) { cfg.Bootstrap.UserPassword = v })

	// Sessions
	applyEnvInt("SESSION_TTL_HOURS", func(v int) { cfg.Session.TTLHours = v })
	applyEnvString("SESSION_SECRET", func(v string) { cfg.Session.Secret = v })
	applyEnvString("SESSION_COOKIE_NAME", func(v string) { cfg.Session.CookieName = v })

	// AI workflow planner
	applyEnvBool("AI_WORKFLOW_GENKIT_ENABLED", func(v bool) { cfg.AIWorkflow.GenkitEnabled = v })
	applyEnvString("AI_WORKFLOW_MODEL", func(v string) { cfg.AIWorkflow.Model = v })
	applyEnvInt("AI_WORKFLOW_TIMEOUT_MS", func(v int) { cfg.AIWorkflow.TimeoutMs = v })
	applyEnvString("AI_WORKFLOW_API_KEY", func(v string) { cfg.AIWorkflow.APIKey = v })
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
		"AI_WORKFLOW_GENKIT_ENABLED",
		"AI_WORKFLOW_MODEL",
		"AI_WORKFLOW_TIMEOUT_MS",
		"AI_WORKFLOW_API_KEY",
	}

	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}
