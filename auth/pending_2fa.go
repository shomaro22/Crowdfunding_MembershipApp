package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	Pending2FACookie = "2fa_pending"
	pending2FATTL    = 5 * time.Minute
)

func SignPending2FA(secret []byte, userID uuid.UUID, now time.Time) string {
	exp := now.Add(pending2FATTL).Unix()
	payload := userID.String() + "|" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

func VerifyPending2FA(secret []byte, token string, now time.Time) (uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return uuid.Nil, ErrInvalidToken
	}
	uidStr, expStr, sig := parts[0], parts[1], parts[2]
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return uuid.Nil, ErrInvalidToken
	}
	payload := uidStr + "|" + expStr
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return uuid.Nil, ErrInvalidToken
	}
	return uuid.Parse(uidStr)
}

func SetPending2FA(c echo.Context, cfg CookieCfg, secret []byte, userID uuid.UUID) {
	val := SignPending2FA(secret, userID, time.Now().UTC())
	c.SetCookie(&http.Cookie{
		Name:     Pending2FACookie,
		Value:    val,
		Path:     cfg.Path,
		MaxAge:   int(pending2FATTL.Seconds()),
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
