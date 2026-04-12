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
	if v, ok := envString("APP_NAME"); ok {
		cfg.AppName = v
	}
	if v, ok := envString("APP_ENV"); ok {
		cfg.AppEnv = AppEnv(v)
	}
	if v, ok := envInt("PORT"); ok {
		cfg.AppPort = v
	}
	if v, ok := envString("APP_TLS_MODE"); ok {
		cfg.AppTLSMode = TLSMode(v)
	}
	if v, ok := envString("APP_TLS_CERT_FILE"); ok {
		cfg.AppTLSCertFile = v
	}
	if v, ok := envString("APP_TLS_KEY_FILE"); ok {
		cfg.AppTLSKeyFile = v
	}
	if v, ok := envString("CSP_EXTRA_DIRECTIVES"); ok {
		cfg.CSPExtraDirectives = v
	}

	if v, ok := envString("DB_TYPE"); ok {
		cfg.DBType = v
	}
	if v, ok := envString("DB_HOST"); ok {
		cfg.DBHost = v
	}
	if v, ok := envString("DB_PORT"); ok {
		cfg.DBPort = v
	}
	if v, ok := envString("DB_USER"); ok {
		cfg.DBUser = v
	}
	if v, ok := envString("DB_PASSWORD"); ok {
		cfg.DBPass = v
	}
	if v, ok := envString("DB_NAME"); ok {
		cfg.DBName = v
	}
	if v, ok := envString("DB_SSL_MODE"); ok {
		cfg.DBSSLMode = v
	}
	if v, ok := envString("DB_TIMEZONE"); ok {
		cfg.DBTimezone = v
	}
	if v, ok := envInt("DB_MAX_IDLE_CONN"); ok {
		cfg.DBMaxIdleConn = v
	}
	if v, ok := envInt("DB_MAX_OPEN_CONN"); ok {
		cfg.DBMaxOpenConn = v
	}
	if v, ok := envInt("DB_CONN_MAX_LIFETIME"); ok {
		cfg.DBConnMaxLifetime = v
	}
	if v, ok := envInt("DB_CONN_MAX_IDLE_TIME"); ok {
		cfg.DBConnMaxIdleTime = v
	}

	if v, ok := envString("REDIS_URL"); ok {
		cfg.RedisURL = v
	}
	if v, ok := envString("REDIS_USERNAME"); ok {
		cfg.RedisUsername = v
	}
	if v, ok := envString("REDIS_PASSWORD"); ok {
		cfg.RedisPassword = v
	}
	if v, ok := envInt("REDIS_DB"); ok {
		cfg.RedisDB = v
	}

	if v, ok := envString("CACHE_URL"); ok {
		cfg.CacheURL = v
	}
	if v, ok := envString("CACHE_USERNAME"); ok {
		cfg.CacheUsername = v
	}
	if v, ok := envString("CACHE_PASSWORD"); ok {
		cfg.CachePassword = v
	}
	if v, ok := envInt("CACHE_DB"); ok {
		cfg.CacheDB = v
	}

	if v, ok := envInt("SUMMARY_CONCURRENCY"); ok {
		cfg.SummaryConcurrency = v
	}
	if v, ok := envInt("SUMMARY_TIMEOUT_MS"); ok {
		cfg.SummaryTimeoutMs = v
	}
	if v, ok := envInt("SUMMARY_REFRESH_INTERVAL_SEC"); ok {
		cfg.SummaryRefreshIntervalSec = v
	}
	if v, ok := envInt("SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC"); ok {
		cfg.SubscriptionClosePeriodIntervalSec = v
	}
	if v, ok := envInt("SUBSCRIPTION_CLOSE_PERIOD_BATCH_SIZE"); ok {
		cfg.SubscriptionClosePeriodBatchSize = v
	}
	if v, ok := envInt("RATING_JOB_INTERVAL_SEC"); ok {
		cfg.RatingJobIntervalSec = v
	}
	if v, ok := envInt("RATING_JOB_BATCH_SIZE"); ok {
		cfg.RatingJobBatchSize = v
	}
	if v, ok := envString("APPS_CREDENTIALS_KEY"); ok {
		cfg.AppsCredentialsKey = v
	}
	if v, ok := envString("STRIPE_CONNECT_CLIENT_ID"); ok {
		cfg.StripeConnectClientID = v
	}
	if v, ok := envString("STRIPE_CONNECT_SECRET"); ok {
		cfg.StripeConnectSecret = v
	}
	if v, ok := envString("STRIPE_CONNECT_REDIRECT_URL"); ok {
		cfg.StripeConnectRedirectURL = v
	}
	if v, ok := envString("PUBLIC_LINK_SECRET"); ok {
		cfg.PublicLinkSecret = v
	}
	if v, ok := envInt("PUBLIC_LINK_TTL_HOURS"); ok {
		cfg.PublicLinkTTLHours = v
	}
	if v, ok := envString("PUBLIC_LINK_BASE_URL"); ok {
		cfg.PublicLinkBaseURL = v
	}

	if v, ok := envBool("ENSURE_DEFAULT_ORG_AND_USER"); ok {
		cfg.BootstrapConfig.EnsureDefaultOrgAndUser = v
	}
	if v, ok := envString("RAILZWAY_ORG_NAME"); ok {
		cfg.BootstrapConfig.OrgName = v
	}
	if v, ok := envString("RAILZWAY_USER_EMAIL"); ok {
		cfg.BootstrapConfig.UserEmail = v
	}
	if v, ok := envString("RAILZWAY_USER_PASSWORD"); ok {
		cfg.BootstrapConfig.UserPassword = v
	}
	if v, ok := envInt("SESSION_TTL_HOURS"); ok {
		cfg.SessionConfig.SessionTTLHours = v
	}
	if v, ok := envString("SESSION_SECRET"); ok {
		cfg.SessionConfig.SessionSecret = v
	}
	if v, ok := envString("SESSION_COOKIE_NAME"); ok {
		cfg.SessionConfig.SessionCookie = v
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
