package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	JWTSecret    string
	Port         string
	PublicRoot   string
	PrivateRoot  string
	APIPublicURL string
	AppPublicURL string
	TOTPIssuer   string
}

func LoadConfig() (Config, error) {
	var (
		cfg Config
		err error
	)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
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
		return Config{}, err
	}
	err = os.MkdirAll(filepath.Join(cfg.PrivateRoot, "kyc"), 0o700)
	if err != nil {
		return Config{}, err
	}
	cfg.APIPublicURL = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if cfg.APIPublicURL == "" {
		cfg.APIPublicURL = "http://localhost:" + cfg.Port
	}
	cfg.AppPublicURL = os.Getenv("PUBLIC_APP_URL")
	cfg.TOTPIssuer = strings.TrimSpace(os.Getenv("TOTP_ISSUER"))
	return cfg, nil
}

func (c Config) ListenAddr() string {
	return ":" + c.Port
}
