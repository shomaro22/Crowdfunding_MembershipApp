package router

import (
	"crowdfund-app/controller"
	"crowdfund-app/middleware"

	"github.com/labstack/echo/v4"
)

type Config struct {
	Health            *controller.HealthController
	PreAuthMiddleware []echo.MiddlewareFunc
}

func New(cfg Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	for _, mw := range cfg.PreAuthMiddleware {
		e.Use(mw)
	}
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())

	if cfg.Health != nil {
		e.GET("/health", cfg.Health.Health)
	}

	return e
}
