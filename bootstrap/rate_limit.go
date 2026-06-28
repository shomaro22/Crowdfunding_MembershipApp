package bootstrap

import (
	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func WireRateLimitActivity(app *App, limiter *middleware.RateLimiter) {
	if app == nil || limiter == nil {
		return
	}
	limiter.OnDenied = func(c echo.Context, policy middleware.RatePolicy, ruleKey string) {
		var (
			uid    *uuid.UUID
			userID uuid.UUID
			ok     bool
			meta   usecase.RequestMeta
		)
		if userID, ok = middleware.AuthUserID(c); ok {
			uid = &userID
		}
		meta = httpRequestMeta(c)
		usecase.LogRateLimitDenied(c.Request().Context(), app.Repos.Activity, uid, policy.Name, ruleKey, meta)
	}
}

func httpRequestMeta(c echo.Context) usecase.RequestMeta {
	var reqID string
	reqID, _ = c.Get(middleware.CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}
