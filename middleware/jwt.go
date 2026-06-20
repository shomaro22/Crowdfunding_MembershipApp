package middleware

import (
	"strings"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	ContextUserIDKey   = "userID"
	ContextUserRoleKey = "role"
)

type TokenParser func(raw string) (auth.AuthClaims, error)

type SessionValidator func(c echo.Context, claims auth.AuthClaims) error

func BearerToken(c echo.Context) string {
	h := c.Request().Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func AuthUserID(c echo.Context) (uuid.UUID, bool) {
	v := c.Get(ContextUserIDKey)
	if v == nil {
		return uuid.Nil, false
	}
	switch x := v.(type) {
	case uuid.UUID:
		return x, x != uuid.Nil
	case string:
		u, err := uuid.Parse(strings.TrimSpace(x))
		return u, err == nil && u != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

func AuthUserRole(c echo.Context) (entity.UserRole, bool) {
	r, ok := c.Get(ContextUserRoleKey).(string)
	if !ok || r == "" {
		return "", false
	}
	return entity.UserRole(r), true
}

func JWTAuth(parse TokenParser, validate SessionValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := BearerToken(c)
			claims, err := parse(raw)
			if err != nil {
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
			}
			c.Set(ContextUserIDKey, claims.UserID)
			c.Set(ContextUserRoleKey, string(claims.Role))
			if validate != nil {
				if err := validate(c, claims); err != nil {
					return err
				}
			}
			return next(c)
		}
	}
}
