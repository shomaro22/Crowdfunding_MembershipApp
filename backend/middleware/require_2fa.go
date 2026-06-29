package middleware

import (
	"github.com/labstack/echo/v4"
)

const CtxTwoFAVerified = "2fa_verified"

func Require2FA() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				v  bool
				ok bool
			)
			v, ok = c.Get(CtxTwoFAVerified).(bool)
			if !ok || !v {
				return echo.NewHTTPError(403, map[string]string{"error": "2fa required", "code": "2FA_REQUIRED"})
			}
			return next(c)
		}
	}
}
