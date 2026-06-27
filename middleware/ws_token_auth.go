package middleware

import (
	"context"
	"strings"

	"crowdfund-app/apierr"
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
				return echo.NewHTTPError(401, apierr.Unauthorized())
			}
			meta = wsTokenMeta(c)
			userID, err = consume(c.Request().Context(), plain, meta)
			if err != nil {
				if apierr.IsTokenAuthFailure(err) {
					return echo.NewHTTPError(401, apierr.Unauthorized())
				}
				return echo.NewHTTPError(401, apierr.Unauthorized())
			}
			c.Set(CtxUserID, userID)
			return next(c)
		}
	}
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
