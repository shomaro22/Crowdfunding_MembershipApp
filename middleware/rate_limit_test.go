package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type denyRateLimitRepo struct{}

func (denyRateLimitRepo) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, int, error) {
	return false, 30, nil
}

func TestRateLimiterOnDenied(t *testing.T) {
	var (
		e       *echo.Echo
		limiter *middleware.RateLimiter
		called  bool
		policy  string
		key     string
		rec     *httptest.ResponseRecorder
		req     *http.Request
	)
	e = echo.New()
	limiter = middleware.NewRateLimiter(&usecase.RateLimitUseCase{Repo: denyRateLimitRepo{}})
	limiter.OnDenied = func(c echo.Context, p middleware.RatePolicy, ruleKey string) {
		called = true
		policy = p.Name
		key = ruleKey
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	e.POST("/auth/login", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, limiter.Middleware(middleware.PolicyLogin))
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429", rec.Code)
	}
	if !called {
		t.Fatal("OnDenied was not called")
	}
	if policy != "login" {
		t.Fatalf("policy=%q want login", policy)
	}
	if key == "" {
		t.Fatal("expected rule key")
	}
}

func TestRateLimiterOnDeniedNilHook(t *testing.T) {
	var (
		e       *echo.Echo
		limiter *middleware.RateLimiter
		rec     *httptest.ResponseRecorder
		req     *http.Request
	)
	e = echo.New()
	limiter = middleware.NewRateLimiter(&usecase.RateLimitUseCase{Repo: denyRateLimitRepo{}})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	e.POST("/auth/login", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, limiter.Middleware(middleware.PolicyLogin))
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429", rec.Code)
	}
}
