package stripe_test

import (
	"testing"

	"crowdfund-app/stripe"
)

func TestConfigFromEnv_NotConfigured(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "")
	_, err := stripe.ConfigFromEnv()
	if err != stripe.ErrNotConfigured {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

func TestConfigFromEnv_DefaultConnectURLs(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_x")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_x")
	t.Setenv("PUBLIC_APP_URL", "http://localhost:3000")
	t.Setenv("STRIPE_CONNECT_RETURN_URL", "")
	t.Setenv("STRIPE_CONNECT_REFRESH_URL", "")

	cfg, err := stripe.ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnectReturn != "http://localhost:3000/seller/connect/return" {
		t.Fatalf("return URL: %q", cfg.ConnectReturn)
	}
	if cfg.ConnectRefresh != "http://localhost:3000/seller/connect/refresh" {
		t.Fatalf("refresh URL: %q", cfg.ConnectRefresh)
	}
}
