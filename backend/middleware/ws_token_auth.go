package middleware

import (
	"context"
	"errors"
	"strings"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type WSTokenConsumer func(ctx context.Context, plain string, meta usecase.RequestMeta) (uuid.UUID, error)

func WSTokenAuth(consume WSTokenConsumer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				plain  string
				userID uuid.UUID
				meta   usecase.RequestMeta
				err    error
			)
			plain = strings.TrimSpace(c.QueryParam("ws_token"))
			if plain == "" {
				return unauthorizedHTTPError()
			}
			meta = wsTokenMeta(c)
			userID, err = consume(c.Request().Context(), plain, meta)
			if err != nil {
				if isTokenAuthFailure(err) {
					return unauthorizedHTTPError()
				}
				return unauthorizedHTTPError()
			}
			c.Set(CtxUserID, userID)
			return next(c)
		}
	}
}

func isTokenAuthFailure(err error) bool {
	return errors.Is(err, usecase.ErrInvalidToken) ||
		errors.Is(err, usecase.ErrForbidden) ||
		errors.Is(err, usecase.ErrInactiveUser)
}

func unauthorizedHTTPError() *echo.HTTPError {
	return echo.NewHTTPError(401, map[string]string{
		"error": "unauthorized",
		"code":  "UNAUTHORIZED",
	})
}

func wsTokenMeta(c echo.Context) usecase.RequestMeta {
	var reqID string
	reqID, _ = c.Get(CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}
