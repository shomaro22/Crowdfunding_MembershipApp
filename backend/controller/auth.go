package controller

import (
	"net/http"
	"strings"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type authController struct {
	Auth         usecase.IAuthUsecase
	CookieCfg    CookieCfg
	AppPublicURL string
}

func NewAuthController(authUC usecase.IAuthUsecase, cookieCfg CookieCfg, appPublicURL string) IAuthController {
	return &authController{
		Auth:         authUC,
		CookieCfg:    cookieCfg,
		AppPublicURL: appPublicURL,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePwRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	TotpCode        string `json:"totp_code"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *authController) Login(c echo.Context) error {
	var (
		body  loginRequest
		res   *usecase.LoginResult
		token string
		err   error
	)
	if err = bindBody(c, &body); err != nil {
		return err
	}
	res, err = h.Auth.Login(reqCtx(c), body.Email, body.Password, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	if res.Requires2FA {
		token, err = h.Auth.CreatePending2FAToken(reqCtx(c), res.User.ID)
		if err != nil {
			return mapUseCaseError(err)
		}
		SetPending2FA(c, h.CookieCfg, token)
		return jsonOK(c, login2FAResponse{
			Requires2FA: true,
			User:        UserWireFrom(res.User),
		})
	}
	SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return jsonOK(c, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}

func (h *authController) Refresh(c echo.Context) error {
	var (
		plain        string
		authToken    string
		refreshToken string
		err          error
	)
	plain = RefreshTokenFrom(c, h.CookieCfg)
	if strings.TrimSpace(plain) == "" {
		return c.NoContent(http.StatusNoContent)
	}
	authToken, refreshToken, err = h.Auth.Refresh(reqCtx(c), plain, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	SetRefreshCookie(c, h.CookieCfg, refreshToken)
	return jsonOK(c, map[string]string{"auth_token": authToken})
}

func (h *authController) Logout(c echo.Context) error {
	var (
		plain string
		err   error
	)
	plain = RefreshTokenFrom(c, h.CookieCfg)
	if err = h.Auth.Logout(reqCtx(c), plain, requestMeta(c)); err != nil {
		return mapUseCaseError(err)
	}
	ClearRefreshCookie(c, h.CookieCfg)
	return noContent(c)
}

func (h *authController) LogoutAllDevices(c echo.Context) error {
	var (
		userID uuid.UUID
		ok     bool
		err    error
	)
	userID, ok = middleware.AuthUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
	}
	if err = h.Auth.LogoutAllDevices(reqCtx(c), userID, requestMeta(c)); err != nil {
		return mapUseCaseError(err)
	}
	ClearRefreshCookie(c, h.CookieCfg)
	return noContent(c)
}

func (h *authController) ForgotPassword(c echo.Context) error {
	var (
		body forgotPasswordRequest
		err  error
	)
	if err = bindBody(c, &body); err != nil {
		return err
	}
	_ = h.Auth.ForgotPassword(reqCtx(c), body.Email)
	return noContent(c)
}

func (h *authController) ResetPassword(c echo.Context) error {
	var (
		body resetPasswordRequest
		err  error
	)
	if err = bindBody(c, &body); err != nil {
		return err
	}
	if err = h.Auth.ResetPassword(reqCtx(c), body.Token, body.Password); err != nil {
		return mapUseCaseError(err)
	}
	return noContent(c)
}

func (h *authController) Me(c echo.Context) error {
	var (
		uid uuid.UUID
		u   usecase.UserView
		err error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	u, err = h.Auth.GetUser(reqCtx(c), uid)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]UserWire{"user": UserWireFrom(u)})
}

func (h *authController) ChangePassword(c echo.Context) error {
	var (
		uid  uuid.UUID
		body changePwRequest
		err  error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	if err = h.Auth.ChangePassword(reqCtx(c), uid, body.CurrentPassword, body.NewPassword, body.TotpCode); err != nil {
		return mapUseCaseError(err)
	}
	return noContent(c)
}

func (h *authController) CheckEmail(c echo.Context) error {
	var (
		email     string
		available bool
		err       error
	)
	email = c.QueryParam("email")
	available, err = h.Auth.EmailFree(reqCtx(c), email)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]bool{"available": available})
}

func (h *authController) GetProfile(c echo.Context) error {
	var (
		uid uuid.UUID
		u   usecase.UserView
		err error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	u, err = h.Auth.GetUser(reqCtx(c), uid)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, UserWireFrom(u))
}

func (h *authController) UpdateProfile(c echo.Context) error {
	var (
		uid  uuid.UUID
		body updateProfileJSON
		u    usecase.UserView
		err  error
	)
	uid, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	u, err = h.Auth.UpdateProfile(reqCtx(c), uid, usecase.UpdateProfileIn{
		Name:      body.Name,
		AvatarURL: body.AvatarURL,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, UserWireFrom(u))
}
