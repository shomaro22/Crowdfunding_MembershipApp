package middleware

import (
	"crowdfund-app/model"

	"github.com/labstack/echo/v4"
)

const CtxAdminRole = "admin_role"

func AuthAdminRole(c echo.Context) (model.AdminRole, bool) {
	var (
		r  string
		ar model.AdminRole
		ok bool
	)
	r, ok = c.Get(CtxAdminRole).(string)
	if !ok || r == "" {
		return "", false
	}
	ar = model.AdminRole(r)
	if !ar.Valid() {
		return "", false
	}
	return ar, true
}

func setAdminRoleCtx(c echo.Context, role *model.AdminRole) {
	if role == nil {
		return
	}
	c.Set(CtxAdminRole, string(*role))
}

// SetAdminRoleFromSession sets admin_role when the validated session belongs to an admin.
func SetAdminRoleFromSession(c echo.Context, role model.UserRole, adminRole *model.AdminRole) {
	if role != model.UserRoleAdmin {
		return
	}
	setAdminRoleCtx(c, adminRole)
}

// RequireAdminRole checks JWT admin_role against allowed roles (§8.6).
// SUPER_ADMIN is always permitted.
func RequireAdminRole(roles ...model.AdminRole) echo.MiddlewareFunc {
	var (
		allowed map[model.AdminRole]struct{}
		i       int
		r       model.AdminRole
	)
	allowed = make(map[model.AdminRole]struct{}, len(roles))
	for i = range roles {
		r = roles[i]
		if r.Valid() {
			allowed[r] = struct{}{}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				role model.UserRole
				ar   model.AdminRole
				ok   bool
			)
			role, ok = AuthUserRole(c)
			if !ok || role != model.UserRoleAdmin {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			ar, ok = AuthAdminRole(c)
			if !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			if ar == model.AdminRoleSuperAdmin {
				return next(c)
			}
			if _, ok = allowed[ar]; !ok {
				return echo.NewHTTPError(403, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
			}
			return next(c)
		}
	}
}
