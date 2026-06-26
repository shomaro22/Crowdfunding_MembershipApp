package auth

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	DefaultRefreshCookie = "refresh_token"
	refreshTokenMaxAge   = 7 * 24 * 3600
)

type CookieCfg struct {
	Name     string
	Secure   bool
	SameSite http.SameSite
	Path     string
}

func CookieFromEnv() CookieCfg {
	var (
		name   string
		secure bool
	)
	name = strings.TrimSpace(os.Getenv("REFRESH_COOKIE_NAME"))
	if name == "" {
		name = DefaultRefreshCookie
	}
	secure = strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
	return CookieCfg{
		Name:     name,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
}

func SetRefreshCookie(c echo.Context, cfg CookieCfg, value string) {
	c.SetCookie(&http.Cookie{
		Name:     cfg.Name,
		Value:    value,
		Path:     cfg.Path,
		MaxAge:   refreshTokenMaxAge,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

func ClearRefreshCookie(c echo.Context, cfg CookieCfg) {
	c.SetCookie(&http.Cookie{
		Name:     cfg.Name,
		Value:    "",
		Path:     cfg.Path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Expires:  time.Unix(0, 0),
	})
}

func RefreshTokenFrom(c echo.Context, cfg CookieCfg) string {
	var (
		ck  *http.Cookie
		err error
	)
	ck, err = c.Cookie(cfg.Name)
	if err != nil || ck == nil {
		return ""
	}
	return strings.TrimSpace(ck.Value)
}
