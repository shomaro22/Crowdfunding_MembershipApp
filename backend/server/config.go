package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	JWTSecret     string
	Pending2FAKey []byte
	Port          string
	PublicRoot    string
	PrivateRoot   string
	APIPublicURL  string
	AppPublicURL  string
	TOTPIssuer    string
}

func loadConfig() (config, error) {
	var (
		cfg config
		err error
	)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if len(cfg.JWTSecret) < 32 {
		return config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	pendingKey := strings.TrimSpace(os.Getenv("PENDING_2FA_KEY"))
	if len(pendingKey) >= 32 {
		cfg.Pending2FAKey = []byte(pendingKey)
	} else {
		cfg.Pending2FAKey = []byte(cfg.JWTSecret)
	}
	cfg.Port = os.Getenv("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	cfg.PublicRoot = os.Getenv("PUBLIC_ROOT")
	if cfg.PublicRoot == "" {
		cfg.PublicRoot = "data/public"
	}
	cfg.PrivateRoot = os.Getenv("PRIVATE_ROOT")
	if cfg.PrivateRoot == "" {
		cfg.PrivateRoot = "data/private"
	}
	err = os.MkdirAll(filepath.Join(cfg.PublicRoot, "uploads"), 0o755)
	if err != nil {
		return config{}, err
	}
	err = os.MkdirAll(filepath.Join(cfg.PrivateRoot, "kyc"), 0o700)
	if err != nil {
		return config{}, err
	}
	cfg.APIPublicURL = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if cfg.APIPublicURL == "" {
		cfg.APIPublicURL = "http://localhost:" + cfg.Port
	}
	cfg.AppPublicURL = os.Getenv("PUBLIC_APP_URL")
	cfg.TOTPIssuer = strings.TrimSpace(os.Getenv("TOTP_ISSUER"))
	return cfg, nil
}

func (c config) listenAddr() string {
	return ":" + c.Port
}

func databaseURL() string {
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
}
