package middleware

import (
	"crowdfund-app/entity"

	"github.com/labstack/echo/v4"
)

func RequireRole(roles ...entity.UserRole) echo.MiddlewareFunc {
	var (
		allowed map[entity.UserRole]struct{}
		i       int
		r       entity.UserRole
	)
	allowed = make(map[entity.UserRole]struct{}, len(roles))
	for i = range roles {
		r = roles[i]
		allowed[r] = struct{}{}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				role entity.UserRole
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
