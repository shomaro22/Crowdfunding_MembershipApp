package middleware

import (
	"context"

	"crowdfund-app/apierr"
	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type SessionChecker func(ctx context.Context, claims auth.AuthClaims) (usecase.SessionInfo, error)

func SessionValidateHTTP(check SessionChecker) SessionValidator {
	return func(c echo.Context, claims auth.AuthClaims) error {
		var (
			info usecase.SessionInfo
			err  error
		)
		info, err = check(c.Request().Context(), claims)
		if err != nil {
			var (
				status int
				body   apierr.Body
			)
			status, body = apierr.MapSession(err)
			return echo.NewHTTPError(status, body)
		}
		c.Set(CtxUserRole, string(info.Role))
		SetAdminRoleFromSession(c, info.Role, info.AdminRole)
		return nil
	}
}
