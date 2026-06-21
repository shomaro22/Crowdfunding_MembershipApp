package controller

import (
	"net/http"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

func (h *AuthController) Setup2FA(c echo.Context) error {
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	res, err := h.Auth.Setup2FA(c.Request().Context(), uid)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"secret":      res.Secret,
		"otpauth_url": res.OTPAuthURL,
	})
}

func (h *AuthController) Verify2FA(c echo.Context) error {
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err := h.Auth.Verify2FASetup(c.Request().Context(), uid, body.Code)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"enabled":      true,
		"backup_codes": res.BackupCodes,
	})
}

func (h *AuthController) Challenge2FA(c echo.Context) error {
	pending := auth.Pending2FAFrom(c)
	if pending == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("2fa session expired", "UNAUTHORIZED"))
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err := h.Auth.Challenge2FA(c.Request().Context(), pending, body.Code, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	auth.ClearPending2FA(c, h.CookieCfg)
	auth.SetReCookie(c, h.CookieCfg, res.ReToken)
	return c.JSON(http.StatusOK, map[string]any{
		"auth_token": res.AuthToken,
		"user":       userToJSON(res.User),
	})
}

func (h *AuthController) GoogleStart(c echo.Context) error {
	from := c.QueryParam("from")
	authURL, err := h.Auth.GoogleAuthURL(from)
	if err != nil {
		return mapAuthError(err)
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (h *AuthController) GoogleCallback(c echo.Context) error {
	if errMsg := c.QueryParam("error"); errMsg != "" {
		return c.Redirect(http.StatusFound, h.Auth.OAuthErrorRedirect(errMsg))
	}
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	res, err := h.Auth.GoogleCallback(c.Request().Context(), code, state, authMeta(c))
	if err != nil {
		return c.Redirect(http.StatusFound, h.Auth.OAuthErrorRedirect(oauthErrCode(err)))
	}
	auth.SetReCookie(c, h.CookieCfg, res.ReToken)
	return c.Redirect(http.StatusFound, h.Auth.OAuthSuccessRedirect(res.From, res.CallbackCode))
}

func (h *AuthController) OAuthExchange(c echo.Context) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err := h.Auth.ExchangeOAuthCode(c.Request().Context(), body.Code, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"auth_token": res.AuthToken,
		"user":       userToJSON(res.User),
	})
}

func oauthErrCode(err error) string {
	switch err {
	case usecase.ErrInactiveUser:
		return "user_inactive"
	case usecase.ErrOAuthForbidden:
		return "oauth_forbidden"
	default:
		return "oauth_failed"
	}
}
