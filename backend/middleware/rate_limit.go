package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type RateRule struct {
	Limit int
	Key   func(c echo.Context) string
}

type RatePolicy struct {
	Name   string
	Period time.Duration
	Rules  []RateRule
}

type RateLimiter struct {
	UC       usecase.IRateLimitUsecase
	OnDenied RateLimitDeniedHook
}

type RateLimitDeniedHook func(c echo.Context, policy RatePolicy, ruleKey string)

func NewRateLimiter(uc usecase.IRateLimitUsecase) *RateLimiter {
	return &RateLimiter{UC: uc}
}

func (l *RateLimiter) Middleware(policy RatePolicy) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				ctx   context.Context
				i     int
				rule  RateRule
				key   string
				full  string
				ok    bool
				retry int
				err   error
			)
			if l == nil || l.UC == nil {
				return next(c)
			}
			ctx = c.Request().Context()
			for i = 0; i < len(policy.Rules); i++ {
				rule = policy.Rules[i]
				key = rule.Key(c)
				if key == "" || rule.Limit <= 0 {
					continue
				}
				full = policy.Name + ":" + key
				ok, retry, err = l.UC.Allow(ctx, full, rule.Limit, policy.Period)
				if err != nil {
					slog.Warn("rate_limit_fail_open", "policy", policy.Name, "key", full, "error", err.Error())
					return next(c)
				}
				if !ok {
					if l.OnDenied != nil {
						l.OnDenied(c, policy, full)
					}
					c.Response().Header().Set("Retry-After", strconv.Itoa(retry))
					return rateLimited(retry)
				}
			}
			return next(c)
		}
	}
}

type rateLimitErrorBody struct {
	Error             string `json:"error"`
	Code              string `json:"code"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

func rateLimited(retryAfterSec int) *echo.HTTPError {
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	return echo.NewHTTPError(http.StatusTooManyRequests, rateLimitErrorBody{
		Error:             "rate limit exceeded",
		Code:              "RATE_LIMITED",
		RetryAfterSeconds: retryAfterSec,
	})
}
