package oauth2provider

import (
	"time"

	"github.com/smallbiznis/railzway/internal/config"
)

// Config holds OAuth2 provider configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	CodeTTL      time.Duration
	AccessTTL    time.Duration
}

func NewConfig(cfg config.Config) Config {
	return Config{
		ClientID:     cfg.OAuth2ClientID,
		ClientSecret: cfg.OAuth2ClientSecret,
		CodeTTL:      5 * time.Minute,
		AccessTTL:    time.Hour,
	}
}
