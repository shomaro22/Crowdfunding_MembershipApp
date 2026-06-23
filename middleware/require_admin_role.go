package middleware

import (
	"crowdfund-app/entity"

	"github.com/labstack/echo/v4"
)

const CtxAdminRole = "admin_role"

func AuthAdminRole(c echo.Context) (entity.AdminRole, bool) {
	var (
		r  string
		ar entity.AdminRole
		ok bool
	)
	r, ok = c.Get(CtxAdminRole).(string)
	if !ok || r == "" {
		return "", false
	}
	ar = entity.AdminRole(r)
	if !ar.Valid() {
		return "", false
	}
	return ar, true
}

func setAdminRoleCtx(c echo.Context, role *entity.AdminRole) {
	if role == nil {
		return
	}
	c.Set(CtxAdminRole, string(*role))
}

// SetAdminRoleFromUser refreshes admin_role in context from the DB user (after session validation).
func SetAdminRoleFromUser(c echo.Context, u *entity.User) {
	if u == nil || u.Role != entity.UserRoleAdmin {
		return
	}
	setAdminRoleCtx(c, u.AdminRole)
}

// RequireAdminRole checks JWT admin_role against allowed roles (§8.6).
// SUPER_ADMIN is always permitted.
func RequireAdminRole(roles ...entity.AdminRole) echo.MiddlewareFunc {
	var (
		allowed map[entity.AdminRole]struct{}
		i       int
		r       entity.AdminRole
	)
	allowed = make(map[entity.AdminRole]struct{}, len(roles))
	for i = range roles {
		r = roles[i]
		if r.Valid() {
			allowed[r] = struct{}{}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				role entity.UserRole
				ar   entity.AdminRole
				ok   bool
			)
			role, ok = AuthUserRole(c)
			if !ok || role != entity.UserRoleAdmin {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			ar, ok = AuthAdminRole(c)
			if !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			if ar == entity.AdminRoleSuperAdmin {
				return next(c)
			}
			if _, ok = allowed[ar]; !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			return next(c)
		}
	}
}
