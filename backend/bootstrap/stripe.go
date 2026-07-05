package bootstrap

import (
	"log/slog"
	"os"
	"strings"

	"crowdfund-app/stripe"
)

// StripeFromEnv は環境に応じた Stripe クライアントを返す。
func StripeFromEnv() stripe.Client {
	if strings.TrimSpace(os.Getenv("STRIPE_USE_STUB")) == "1" {
		slog.Info("stripe using stub client for local development")
		return stripe.NewStubClient()
	}
	cfg, err := stripe.ConfigFromEnv()
	if err != nil {
		if isProduction() {
			slog.Info("stripe disabled", "reason", err.Error())
			return nil
		}
		slog.Info("stripe using stub client", "reason", err.Error())
		return stripe.NewStubClient()
	}
	return stripe.New(cfg)
}
