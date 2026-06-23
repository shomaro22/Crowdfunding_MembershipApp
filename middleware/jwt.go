package middleware

import (
	"strings"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	CtxUserID   = "userID"
	CtxUserRole = "role"
)

type TokenParser func(raw string) (auth.AuthClaims, error)

type SessionValidator func(c echo.Context, claims auth.AuthClaims) error

func BearerToken(c echo.Context) string {
	var (
		h string
	)
	h = c.Request().Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func AuthUserID(c echo.Context) (uuid.UUID, bool) {
	var (
		v   interface{}
		x   uuid.UUID
		s   string
		u   uuid.UUID
		err error
	)
	v = c.Get(CtxUserID)
	if v == nil {
		return uuid.Nil, false
	}
	switch v.(type) {
	case uuid.UUID:
		x = v.(uuid.UUID)
		return x, x != uuid.Nil
	case string:
		s = v.(string)
		u, err = uuid.Parse(strings.TrimSpace(s))
		return u, err == nil && u != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

func AuthUserRole(c echo.Context) (entity.UserRole, bool) {
	var (
		r  string
		ok bool
	)
	r, ok = c.Get(CtxUserRole).(string)
	if !ok || r == "" {
		return "", false
	}
	return entity.UserRole(r), true
}

func JWTAuth(parse TokenParser, validate SessionValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				raw    string
				claims auth.AuthClaims
				err    error
			)
			raw = BearerToken(c)
			claims, err = parse(raw)
			if err != nil {
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
			}
			c.Set(CtxUserID, claims.UserID)
			c.Set(CtxUserRole, string(claims.Role))
			c.Set(CtxTwoFAVerified, claims.TwoFAVerified)
			setAdminRoleCtx(c, claims.AdminRole)
			if validate != nil {
				if err = validate(c, claims); err != nil {
					return err
				}
			}
			return next(c)
		}
	}
}

// OptionalJWTAuth sets user context when a valid Bearer token is present; otherwise continues anonymously.
func OptionalJWTAuth(parse TokenParser, validate SessionValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				raw    string
				claims auth.AuthClaims
				err    error
			)
			raw = BearerToken(c)
			if raw == "" {
				return next(c)
			}
			claims, err = parse(raw)
			if err != nil {
				return next(c)
			}
			c.Set(CtxUserID, claims.UserID)
			c.Set(CtxUserRole, string(claims.Role))
			c.Set(CtxTwoFAVerified, claims.TwoFAVerified)
			setAdminRoleCtx(c, claims.AdminRole)
			if validate != nil {
				if err = validate(c, claims); err != nil {
					return next(c)
				}
			}
			return next(c)
		}
	}
}
