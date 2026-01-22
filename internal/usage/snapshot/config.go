package snapshot

import "time"

// Config controls the usage snapshot worker loop.
type Config struct {
	BatchSize    int
	PollInterval time.Duration
	RunTimeout   time.Duration
	RowTimeout   time.Duration
}

func DefaultConfig() Config {
	return Config{
		BatchSize:    50,
		PollInterval: 2 * time.Second,
		RunTimeout:   5 * time.Second,
		RowTimeout:   300 * time.Millisecond,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.BatchSize <= 0 {
		c.BatchSize = defaults.BatchSize
	}

	if c.PollInterval <= 0 {
		c.PollInterval = defaults.PollInterval
	}
	return c
}
