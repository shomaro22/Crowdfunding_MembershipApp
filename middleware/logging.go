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
			id := c.Request().Header.Get("X-Request-ID")
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
			err := next(c)
			reqID, _ := c.Get(CtxRequestID).(string)
			attrs := []any{
				"request_id", reqID,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
			}
			if err != nil {
				attrs = append(attrs, "error", err.Error())
				slog.Error("request", attrs...)
				return err
			}
			slog.Info("request", attrs...)
			return nil
		}
	}
}
