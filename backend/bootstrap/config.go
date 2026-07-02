// Package bootstrap はアプリケーションの composition root（配線）を集約する。
package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SecurityConfig は署名・暗号化用の秘密鍵を保持する（用途ごとに分離）。
type SecurityConfig struct {
	JWTSecret         string
	RefreshSecret     []byte
	TOTPEncryptionKey []byte
	KYCSigningKey     []byte
	Pending2FAKey     []byte
	TOTPIssuer        string
}

// DatabaseConfig は DB 接続設定。
type DatabaseConfig struct {
	URL string
}

// StorageConfig は公開・非公開ファイルストレージのルート。
type StorageConfig struct {
	PublicRoot  string
	PrivateRoot string
}

// OAuthConfig は OAuth リダイレクト用の公開 URL。
type OAuthConfig struct {
	AppPublicURL string
	APIPublicURL string
}

// AppConfig は起動に必要な設定の集約。
type AppConfig struct {
	Security SecurityConfig
	Database DatabaseConfig
	Storage  StorageConfig
	OAuth    OAuthConfig
	Port     string
}

func (c AppConfig) ListenAddr() string {
	return ":" + c.Port
}

func secretBytes(envKey string, fallback string, label string) []byte {
	v := strings.TrimSpace(os.Getenv(envKey))
	if len(v) >= 32 {
		return []byte(v)
	}
	if isProduction() {
		panic(fmt.Sprintf("%s must be at least 32 characters in production", envKey))
	}
	if len(fallback) >= 32 {
		slog.Warn("security: using JWT_SECRET fallback for key; set distinct secret in production", "key", label)
		return []byte(fallback)
	}
	panic(fmt.Sprintf("%s: fallback too short", label))
}

func isProduction() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

// LoadConfig は環境変数から設定を読み込む。
func LoadConfig() (AppConfig, error) {
	var (
		cfg AppConfig
		err error
	)
	cfg.Security.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(cfg.Security.JWTSecret) < 32 {
		return AppConfig{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	cfg.Security.RefreshSecret = secretBytes("REFRESH_SECRET", cfg.Security.JWTSecret, "REFRESH_SECRET")
	cfg.Security.TOTPEncryptionKey = secretBytes("TOTP_ENCRYPTION_KEY", cfg.Security.JWTSecret, "TOTP_ENCRYPTION_KEY")
	cfg.Security.KYCSigningKey = secretBytes("KYC_SIGNING_KEY", cfg.Security.JWTSecret, "KYC_SIGNING_KEY")
	cfg.Security.Pending2FAKey = secretBytes("PENDING_2FA_KEY", cfg.Security.JWTSecret, "PENDING_2FA_KEY")
	cfg.Security.TOTPIssuer = strings.TrimSpace(os.Getenv("TOTP_ISSUER"))

	cfg.Port = strings.TrimSpace(os.Getenv("PORT"))
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	cfg.Storage.PublicRoot = strings.TrimSpace(os.Getenv("PUBLIC_ROOT"))
	if cfg.Storage.PublicRoot == "" {
		cfg.Storage.PublicRoot = "data/public"
	}
	cfg.Storage.PrivateRoot = strings.TrimSpace(os.Getenv("PRIVATE_ROOT"))
	if cfg.Storage.PrivateRoot == "" {
		cfg.Storage.PrivateRoot = "data/private"
	}
	err = os.MkdirAll(filepath.Join(cfg.Storage.PublicRoot, "uploads"), 0o755)
	if err != nil {
		return AppConfig{}, err
	}
	err = os.MkdirAll(filepath.Join(cfg.Storage.PrivateRoot, "kyc"), 0o700)
	if err != nil {
		return AppConfig{}, err
	}

	cfg.Database.URL = DatabaseURL()

	cfg.OAuth.APIPublicURL = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if cfg.OAuth.APIPublicURL == "" {
		cfg.OAuth.APIPublicURL = "http://localhost:" + cfg.Port
	}
	cfg.OAuth.AppPublicURL = strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))

	return cfg, nil
}

// DatabaseURL は DATABASE_URL を返す（未設定時はローカルデフォルト）。
func DatabaseURL() string {
	u := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
}

// CORSOrigins は許可オリジン一覧を返す。
func CORSOrigins() []string {
	var (
		raw     string
		origins []string
		parts   []string
		i       int
		s       string
	)
	raw = strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if raw == "" {
		return nil
	}
	parts = strings.Split(raw, ",")
	for i = range parts {
		s = strings.TrimSpace(parts[i])
		if s != "" {
			origins = append(origins, s)
		}
	}
	return origins
}
