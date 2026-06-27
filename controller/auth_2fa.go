package controller

import (
	"net/http"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

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
