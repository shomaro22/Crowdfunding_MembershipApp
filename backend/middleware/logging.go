package middleware

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const CtxRequestID = "request_id"

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var id string
			id = c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.NewString()
			}
			c.Set(CtxRequestID, id)
			c.Response().Header().Set("X-Request-ID", id)
			return next(c)
		}
	}
}

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				err   error
				reqID string
			)
			err = next(c)
			reqID, _ = c.Get(CtxRequestID).(string)
			if err != nil {
				slog.Error("request",
					slog.String("request_id", reqID),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", c.Response().Status),
					slog.String("error", err.Error()),
				)
				return err
			}
			slog.Info("request",
				slog.String("request_id", reqID),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", c.Response().Status),
			)
			return nil
		}
	}
}
