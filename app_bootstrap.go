package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"crowdfund-app/controller"
	"crowdfund-app/db"
	"crowdfund-app/router"

	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
)

func buildApp() (*echo.Echo, string, error) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	if err := db.Up(databaseURL); err != nil {
		return nil, "", err
	}
	gormDB, err := db.Open(databaseURL)
	if err != nil {
		return nil, "", err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, "", err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, "", err
	}

	rdb := db.OpenRedis()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	publicRoot := os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		publicRoot = "data/public"
	}
	if err := os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755); err != nil {
		return nil, "", err
	}

	healthCtl := &controller.HealthController{SQL: gormDB, RDB: rdb}

	var preAuth []echo.MiddlewareFunc
	corsRaw := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		var origins []string
		for _, p := range strings.Split(corsRaw, ",") {
			if s := strings.TrimSpace(p); s != "" {
				origins = append(origins, s)
			}
		}
		if len(origins) > 0 {
			preAuth = append(preAuth, labmw.CORSWithConfig(labmw.CORSConfig{
				AllowOrigins: origins,
				AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
				AllowMethods: []string{
					http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
					http.MethodPost, http.MethodDelete, http.MethodOptions,
				},
			}))
		}
	}

	e := router.New(router.Config{
		PreAuthMiddleware: preAuth,
		Health:            healthCtl,
	})
	return e, ":" + port, nil
}
