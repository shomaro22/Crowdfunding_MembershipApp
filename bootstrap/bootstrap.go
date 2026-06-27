// Package bootstrap wires shared dependencies for HTTP and batch entrypoints.
package bootstrap

import (
	"log"
	"log/slog"
	"os"

	"crowdfund-app/stripe"

	"github.com/joho/godotenv"
)

func LoadDotEnv() {
	const envFile = ".env"
	var err error
	_, err = os.Stat(envFile)
	if err != nil {
		return
	}
	err = godotenv.Load(envFile)
	if err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}

func DatabaseURL() string {
	var u string
	u = os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
}

func StripeFromEnv() stripe.Client {
	var (
		cfg stripe.Config
		err error
	)
	cfg, err = stripe.ConfigFromEnv()
	if err != nil {
		slog.Info("stripe disabled", "reason", err.Error())
		return nil
	}
	return stripe.New(cfg)
}
