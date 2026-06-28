package controller

import (
	"errors"
	"net/http"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *AuthController) GoogleStart(c echo.Context) error {
	var (
		from    string
		authURL string
		err     error
	)
	from = c.QueryParam("from")
	authURL, err = h.Auth.GoogleAuthURL(from)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (h *AuthController) GoogleCallback(c echo.Context) error {
	var (
		errMsg string
		code   string
		state  string
		res    *usecase.OAuthCBOut
		err    error
	)
	errMsg = c.QueryParam("error")
	if errMsg != "" {
		return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, usecase.OAuthRedirectView{Error: errMsg}))
	}
	code = c.QueryParam("code")
	state = c.QueryParam("state")
	res, err = h.Auth.GoogleCallback(c.Request().Context(), code, state, requestMeta(c))
	if err != nil {
		return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, usecase.OAuthRedirectView{Error: oauthErrCode(err)}))
	}
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, res.Redirect))
}

func (h *AuthController) OAuthExchange(c echo.Context) error {
	var (
		body struct {
			Code string `json:"code"`
		}
		res *usecase.LoginResult
		err error
	)
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.ExchangeOAuthCode(c.Request().Context(), body.Code, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}

func oauthErrCode(err error) string {
	switch {
	case errors.Is(err, usecase.ErrInactiveUser):
		return "user_inactive"
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return "oauth_forbidden"
	default:
		return "oauth_failed"
	}
}

func (h *AuthController) Setup2FA(c echo.Context) error {
	var (
		uid uuid.UUID
		res *usecase.TOTPSetupResult
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	res, err = h.Auth.Setup2FA(c.Request().Context(), uid)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"secret":      res.Secret,
		"otpauth_url": res.OTPAuthURL,
	})
}

func (h *AuthController) Verify2FA(c echo.Context) error {
	var (
		uid  uuid.UUID
		body struct {
			Code string `json:"code"`
		}
		res *usecase.TOTPVerifyResult
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.Verify2FASetup(c.Request().Context(), uid, body.Code)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, verify2FAResponse{
		Enabled:     true,
		BackupCodes: res.BackupCodes,
	})
}

func (h *AuthController) Challenge2FA(c echo.Context) error {
	var (
		pending string
		body    struct {
			Code string `json:"code"`
		}
		res *usecase.LoginResult
		err error
	)
	pending = auth.Pending2FAFrom(c)
	if pending == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("2fa session expired", "UNAUTHORIZED"))
	}
	if err = c.Bind(&body); err != nil || body.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("code required", "BAD_REQUEST"))
	}
	res, err = h.Auth.Challenge2FA(c.Request().Context(), pending, body.Code, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	auth.ClearPending2FA(c, h.CookieCfg)
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}
