package middleware

import (
	"net/http"
	"strconv"
	"time"

	"crowdfund-app/repository"

	"github.com/labstack/echo/v4"
)

type RateRule struct {
	Limit int
	Key   func(c echo.Context) string
}

type RatePolicy struct {
	Name   string
	Window time.Duration
	Rules  []RateRule
}

type RateLimiter struct {
	Repo repository.RateLimitRepo
}

func NewRateLimiter(repo repository.RateLimitRepo) *RateLimiter {
	return &RateLimiter{Repo: repo}
}

func (l *RateLimiter) Middleware(policy RatePolicy) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if l == nil || l.Repo == nil {
				return next(c)
			}
			ctx := c.Request().Context()
			for _, rule := range policy.Rules {
				key := rule.Key(c)
				if key == "" || rule.Limit <= 0 {
					continue
				}
				full := policy.Name + ":" + key
				ok, retry, err := l.Repo.Allow(ctx, full, rule.Limit, policy.Window)
				if err != nil {
					return next(c)
				}
				if !ok {
					c.Response().Header().Set("Retry-After", strconv.Itoa(retry))
					return rateLimited(retry)
				}
			}
			return next(c)
		}
	}
}

func rateLimited(retryAfterSec int) *echo.HTTPError {
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	return echo.NewHTTPError(http.StatusTooManyRequests, map[string]any{
		"error":               "rate limit exceeded",
		"code":                "RATE_LIMITED",
		"retry_after_seconds": retryAfterSec,
	})
}
