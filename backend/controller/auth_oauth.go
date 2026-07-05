package controller

import (
	"errors"
	"net/http"
	"strings"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *authController) GoogleStart(c echo.Context) error {
	var (
		from    string
		authURL string
		err     error
	)
	from = c.QueryParam("from")
	authURL, err = h.Auth.GoogleAuthURL(from, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (h *authController) GoogleCallback(c echo.Context) error {
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
	res, err = h.Auth.GoogleCallback(reqCtx(c), code, state, requestMeta(c))
	if err != nil {
		return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, usecase.OAuthRedirectView{Error: oauthErrCode(err)}))
	}
	if res.RefreshToken != "" {
		SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	}
	return c.Redirect(http.StatusFound, oauthCallbackURL(h.AppPublicURL, res.Redirect))
}

func (h *authController) OAuthExchange(c echo.Context) error {
	var (
		body struct {
			Code string `json:"code"`
		}
		res *usecase.LoginResult
		err error
	)
	if err = bindBody(c, &body); err != nil || body.Code == "" {
		return badRequest(c, "code required")
	}
	res, err = h.Auth.ExchangeOAuthCode(reqCtx(c), body.Code, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}

func (h *authController) OAuthLinkConfirm(c echo.Context) error {
	var (
		body struct {
			LinkCode string `json:"link_code"`
		}
		res *usecase.LoginResult
		err error
	)
	if err = bindBody(c, &body); err != nil || strings.TrimSpace(body.LinkCode) == "" {
		return badRequest(c, "link_code required")
	}
	res, err = h.Auth.ConfirmOAuthLink(reqCtx(c), body.LinkCode, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return jsonOK(c, authTokenResponse{
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

func (h *authController) Setup2FA(c echo.Context) error {
	var (
		uid uuid.UUID
		res *usecase.TOTPSetupResult
		err error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	res, err = h.Auth.Setup2FA(reqCtx(c), uid)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]string{
		"secret":      res.Secret,
		"otpauth_url": res.OTPAuthURL,
	})
}

func (h *authController) Verify2FA(c echo.Context) error {
	var (
		uid  uuid.UUID
		body struct {
			Code string `json:"code"`
		}
		res *usecase.TOTPVerifyResult
		err error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil || body.Code == "" {
		return badRequest(c, "code required")
	}
	res, err = h.Auth.Verify2FASetup(reqCtx(c), uid, body.Code)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, verify2FAResponse{
		Enabled:     true,
		BackupCodes: res.BackupCodes,
	})
}

func (h *authController) Challenge2FA(c echo.Context) error {
	var (
		pending string
		body    struct {
			Code         string `json:"code"`
			PendingToken string `json:"pending_token"`
		}
		res *usecase.LoginResult
		err error
	)
	if err = bindBody(c, &body); err != nil || body.Code == "" {
		return badRequest(c, "code required")
	}
	pending = ResolvePending2FAToken(c, body.PendingToken)
	if pending == "" {
		ClearPending2FA(c, h.CookieCfg)
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("2fa session expired", "UNAUTHORIZED"))
	}
	res, err = h.Auth.Challenge2FA(reqCtx(c), pending, body.Code, requestMeta(c))
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidToken) {
			ClearPending2FA(c, h.CookieCfg)
		}
		return mapUseCaseError(err)
	}
	ClearPending2FA(c, h.CookieCfg)
	SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return jsonOK(c, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}
