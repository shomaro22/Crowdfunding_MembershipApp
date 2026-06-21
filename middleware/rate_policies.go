package middleware

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func KeyIP(c echo.Context) string {
	return c.RealIP()
}

func KeyJSONEmail(field string) func(c echo.Context) string {
	return func(c echo.Context) string {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return ""
		}
		c.Request().Body = io.NopCloser(strings.NewReader(string(body)))
		var m map[string]string
		if json.Unmarshal(body, &m) != nil {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(m[field]))
	}
}

func KeyUserID(c echo.Context) string {
	uid, ok := AuthUserID(c)
	if !ok {
		return ""
	}
	return uid.String()
}

var (
	PolicyLogin = RatePolicy{
		Name:   "login",
		Window: time.Minute,
		Rules: []RateRule{
			{Limit: 20, Key: KeyIP},
			{Limit: 5, Key: KeyJSONEmail("email")},
		},
	}
	PolicyRegisterBuyer = RatePolicy{
		Name:   "register_buyer",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 10, Key: KeyIP}},
	}
	PolicyRegisterSeller = RatePolicy{
		Name:   "register_seller",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 5, Key: KeyIP}},
	}
	PolicyForgotPassword = RatePolicy{
		Name:   "forgot_password",
		Window: time.Minute,
		Rules: []RateRule{
			{Limit: 5, Key: KeyIP},
			{Limit: 3, Key: KeyJSONEmail("email")},
		},
	}
	Policy2FAChallenge = RatePolicy{
		Name:   "2fa/challenge",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 5, Key: KeyIP}},
	}
	PolicyRe = RatePolicy{
		Name:   "re",
		Window: time.Minute,
		Rules: []RateRule{
			{Limit: 30, Key: KeyIP},
			{Limit: 60, Key: KeyUserID},
		},
	}
	PolicyGoogleOAuth = RatePolicy{
		Name:   "google_oauth",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
	PolicyOAuthExchange = RatePolicy{
		Name:   "oauth_exchange",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
)
