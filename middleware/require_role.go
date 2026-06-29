package middleware

import (
	"crowdfund-app/model"

	"github.com/labstack/echo/v4"
)

func RequireRole(roles ...model.UserRole) echo.MiddlewareFunc {
	var (
		allowed map[model.UserRole]struct{}
		i       int
		r       model.UserRole
	)
	allowed = make(map[model.UserRole]struct{}, len(roles))
	for i = range roles {
		r = roles[i]
		allowed[r] = struct{}{}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				role model.UserRole
				ok   bool
			)
			role, ok = AuthUserRole(c)
			if !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			if _, ok = allowed[role]; !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			return next(c)
		}
	}
}
