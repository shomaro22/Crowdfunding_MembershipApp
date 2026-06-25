package stripeclient

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	SecretKey      string
	WebhookSecret  string
	ConnectReturn  string
	ConnectRefresh string
}

func ConfigFromEnv() (Config, error) {
	var (
		cfg Config
		app string
	)
	cfg.SecretKey = strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if cfg.SecretKey == "" {
		return Config{}, ErrNotConfigured
	}
	cfg.WebhookSecret = strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	app = strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_APP_URL")), "/")
	if app == "" {
		app = "http://localhost:5173"
	}
	cfg.ConnectReturn = strings.TrimSpace(os.Getenv("STRIPE_CONNECT_RETURN_URL"))
	if cfg.ConnectReturn == "" {
		cfg.ConnectReturn = app + "/seller/connect/return"
	}
	cfg.ConnectRefresh = strings.TrimSpace(os.Getenv("STRIPE_CONNECT_REFRESH_URL"))
	if cfg.ConnectRefresh == "" {
		cfg.ConnectRefresh = app + "/seller/connect/refresh"
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("%w", ErrNotConfigured)
	}
	return nil
}
