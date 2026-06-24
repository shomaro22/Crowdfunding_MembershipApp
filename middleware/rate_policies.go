package middleware

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"crowdfund-app/auth"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func KeyIP(c echo.Context) string {
	return c.RealIP()
}

func KeyJSONEmail(field string) func(c echo.Context) string {
	return func(c echo.Context) string {
		var (
			body []byte
			err  error
			m    map[string]string
		)
		body, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return ""
		}
		c.Request().Body = io.NopCloser(strings.NewReader(string(body)))
		if json.Unmarshal(body, &m) != nil {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(m[field]))
	}
}

func KeyUserID(c echo.Context) string {
	var (
		uid uuid.UUID
		ok  bool
	)
	uid, ok = AuthUserID(c)
	if !ok {
		return ""
	}
	return uid.String()
}

func KeyPending2FAUser(secret []byte) func(c echo.Context) string {
	return func(c echo.Context) string {
		var (
			tok string
			uid uuid.UUID
			err error
		)
		tok = auth.Pending2FAFrom(c)
		if tok == "" || len(secret) == 0 {
			return ""
		}
		uid, err = auth.VerifyPending2FA(secret, tok, time.Now().UTC())
		if err != nil {
			return ""
		}
		return uid.String()
	}
}

var (
	PolicyLogin = RatePolicy{
		Name:   "login",
		Period: time.Minute,
		Rules: []RateRule{
			{Limit: 20, Key: KeyIP},
			{Limit: 5, Key: KeyJSONEmail("email")},
		},
	}
	PolicyRegisterBuyer = RatePolicy{
		Name:   "register_buyer",
		Period: time.Minute,
		Rules:  []RateRule{{Limit: 10, Key: KeyIP}},
	}
	PolicyRegisterSeller = RatePolicy{
		Name:   "register_seller",
		Period: time.Minute,
		Rules:  []RateRule{{Limit: 5, Key: KeyIP}},
	}
	PolicyForgotPassword = RatePolicy{
		Name:   "forgot_password",
		Period: time.Minute,
		Rules: []RateRule{
			{Limit: 5, Key: KeyIP},
			{Limit: 3, Key: KeyJSONEmail("email")},
		},
	}
	PolicyRe = RatePolicy{
		Name:   "re",
		Period: time.Minute,
		Rules: []RateRule{
			{Limit: 30, Key: KeyIP},
			{Limit: 60, Key: KeyUserID},
		},
	}
)
