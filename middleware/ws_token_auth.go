package middleware

import (
	"context"
	"errors"
	"strings"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func WSTokenAuth(consume func(ctx context.Context, plain string) (uuid.UUID, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				plain  string
				userID uuid.UUID
				err    error
			)
			plain = strings.TrimSpace(c.QueryParam("ws_token"))
			if plain == "" {
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
			}
			userID, err = consume(c.Request().Context(), plain)
			if err != nil {
				if errors.Is(err, usecase.ErrInvalidToken) ||
					errors.Is(err, usecase.ErrForbidden) ||
					errors.Is(err, usecase.ErrInactiveUser) {
					return echo.NewHTTPError(401, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
				}
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
			}
			c.Set(CtxUserID, userID)
			return next(c)
		}
	}
}
