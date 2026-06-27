package middleware

import (
	"context"
	"errors"
	"net/http"

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
			switch {
			case errors.Is(err, usecase.ErrInactiveUser):
				return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "user inactive", "code": "USER_INACTIVE"})
			case errors.Is(err, usecase.ErrTokenRevoked):
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "token revoked", "code": "TOKEN_REVOKED"})
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
			}
		}
		c.Set(CtxUserRole, string(info.Role))
		SetAdminRoleFromSession(c, info.Role, info.AdminRole)
		return nil
	}
}
