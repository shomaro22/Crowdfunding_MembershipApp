package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"crowdfund-app/bootstrap"

	"github.com/joho/godotenv"
)

func LoadDotEnv() {
	const envFile = ".env"
	if _, err := os.Stat(envFile); err != nil {
		return
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}

func main() {
	LoadDotEnv()
	app, err := bootstrap.BuildApplication(nil)
	if err != nil {
		log.Fatalf("startup failed: %v — Render Environment に DATABASE_URL, JWT_SECRET, REFRESH_SECRET, TOTP_ENCRYPTION_KEY, KYC_SIGNING_KEY, PENDING_2FA_KEY を設定してください", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := app.Echo.Start(app.Addr); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	app.Echo.Logger.Info("shutdown server")
	_ = app.Echo.Shutdown(context.Background())
}
