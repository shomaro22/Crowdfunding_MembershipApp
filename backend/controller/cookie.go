package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	DefaultRefreshCookie = "refresh_token"
	Pending2FACookie     = "2fa_pending"
	refreshTokenMaxAge   = 7 * 24 * 3600
)

type CookieCfg struct {
	Name     string
	Secure   bool
	SameSite http.SameSite
	Path     string
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
	ck, err := c.Cookie(cfg.Name)
	if err != nil || ck == nil {
		return ""
	}
	return strings.TrimSpace(ck.Value)
}

func SetPending2FA(c echo.Context, cfg CookieCfg, token string) {
	c.SetCookie(&http.Cookie{
		Name:     Pending2FACookie,
		Value:    token,
		Path:     cfg.Path,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

func ClearPending2FA(c echo.Context, cfg CookieCfg) {
	c.SetCookie(&http.Cookie{
		Name:     Pending2FACookie,
		Value:    "",
		Path:     cfg.Path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Expires:  time.Unix(0, 0),
	})
}

func Pending2FAFrom(c echo.Context) string {
	ck, err := c.Cookie(Pending2FACookie)
	if err != nil || ck == nil {
		return ""
	}
	return strings.TrimSpace(ck.Value)
}

func NewCookieCfg(refreshCookieName string, secure bool) CookieCfg {
	if strings.TrimSpace(refreshCookieName) == "" {
		refreshCookieName = DefaultRefreshCookie
	}
	if secure && !strings.HasPrefix(refreshCookieName, "__Secure-") && !strings.HasPrefix(refreshCookieName, "__Host-") {
		refreshCookieName = "__Secure-" + refreshCookieName
	}

	return CookieCfg{
		Name:     refreshCookieName,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth",
	}
}

// UUIDを利用するControllerとの互換性維持
var _ uuid.UUID