package scheduler

import (
	"os"
	"strings"
	"time"
)

// Config controls scheduler intervals and batch sizes.
type Config struct {
	RunInterval          time.Duration
	BatchSize            int
	RecoveryThreshold    time.Duration
	FinalizeInvoices     bool
	MaxCloseBatchSize    int
	MaxRatingBatchSize   int
	MaxInvoiceBatchSize  int
	WebhookRetentionDays int
	EnabledJobs          []string
}

func ProvideConfig() Config {
	cfg := DefaultConfig()
	if jobs := os.Getenv("ENABLED_JOBS"); jobs != "" {
		cfg.EnabledJobs = strings.Split(jobs, ",")
		for i := range cfg.EnabledJobs {
			cfg.EnabledJobs[i] = strings.TrimSpace(cfg.EnabledJobs[i])
		}
	}
	return cfg
}

func DefaultConfig() Config {
	return Config{
		RunInterval:          time.Minute,
		BatchSize:            50,
		RecoveryThreshold:    15 * time.Minute,
		FinalizeInvoices:     true,
		MaxCloseBatchSize:    50,
		MaxRatingBatchSize:   25,
		MaxInvoiceBatchSize:  25,
		WebhookRetentionDays: 30,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.RunInterval <= 0 {
		c.RunInterval = defaults.RunInterval
	}
	if c.BatchSize <= 0 {
		c.BatchSize = defaults.BatchSize
	}
	if c.RecoveryThreshold <= 0 {
		c.RecoveryThreshold = defaults.RecoveryThreshold
	}
	if c.MaxCloseBatchSize <= 0 {
		c.MaxCloseBatchSize = defaults.MaxCloseBatchSize
	}
	if c.MaxRatingBatchSize <= 0 {
		c.MaxRatingBatchSize = defaults.MaxRatingBatchSize
	}
	if c.MaxInvoiceBatchSize <= 0 {
		c.MaxInvoiceBatchSize = defaults.MaxInvoiceBatchSize
	}
	return c
}
