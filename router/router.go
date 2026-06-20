package router

import (
	"crowdfund-app/controller"
	"crowdfund-app/middleware"

	"github.com/labstack/echo/v4"
)

type Config struct {
	Health            *controller.HealthController
	Auth              *controller.AuthController
	PreAuthMiddleware []echo.MiddlewareFunc
	Protected         []echo.MiddlewareFunc
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

	if cfg.Auth != nil {
		e.POST("/auth/register", cfg.Auth.Register)
		e.POST("/auth/register/seller", cfg.Auth.RegisterSeller)
		e.POST("/auth/login", cfg.Auth.Login)
		e.POST("/auth/forgot-password", cfg.Auth.ForgotPassword)
		e.POST("/auth/reset-password", cfg.Auth.ResetPassword)
		e.GET("/auth/check-email", cfg.Auth.CheckEmail)
		e.POST("/re", cfg.Auth.Re)

		protected := e.Group("", cfg.Protected...)
		protected.POST("/auth/logout", cfg.Auth.Logout)
		protected.GET("/me", cfg.Auth.Me)
		protected.POST("/me/change-password", cfg.Auth.ChangePassword)
	}

	return e
}
