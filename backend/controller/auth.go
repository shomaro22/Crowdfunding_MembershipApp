package controller

import (

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthController struct {
	Auth          usecase.IAuthUsecase
	CookieCfg     auth.CookieCfg
	AppPublicURL  string
	Pending2FAKey []byte
}

func NewAuthController(authUC usecase.IAuthUsecase, cookieCfg auth.CookieCfg, appPublicURL string, pending2FAKey []byte) *AuthController {
	return &AuthController{
		Auth:          authUC,
		CookieCfg:     cookieCfg,
		AppPublicURL:  appPublicURL,
		Pending2FAKey: pending2FAKey,
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

func (h *AuthController) Login(c echo.Context) error {
	var (
		body loginRequest
		res  *usecase.LoginResult
		err  error
	)
	if err = bindBody(c, &body); err != nil {
		return err
	}
	res, err = h.Auth.Login(reqCtx(c), body.Email, body.Password, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	if res.Requires2FA {
		auth.SetPending2FA(c, h.CookieCfg, h.Pending2FAKey, res.User.ID)
		return jsonOK(c, login2FAResponse{
			Requires2FA: true,
			User:        UserWireFrom(res.User),
		})
	}
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return jsonOK(c, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      UserWireFrom(res.User),
	})
}

func (h *AuthController) Refresh(c echo.Context) error {
	var (
		plain        string
		authToken    string
		refreshToken string
		err          error
	)
	plain = auth.RefreshTokenFrom(c, h.CookieCfg)
	authToken, refreshToken, err = h.Auth.Refresh(reqCtx(c), plain, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	auth.SetRefreshCookie(c, h.CookieCfg, refreshToken)
	return jsonOK(c, map[string]string{"auth_token": authToken})
}

func (h *AuthController) Logout(c echo.Context) error {
	var (
		plain string
		err   error
	)
	plain = auth.RefreshTokenFrom(c, h.CookieCfg)
	if err = h.Auth.Logout(reqCtx(c), plain, requestMeta(c)); err != nil {
		return mapUseCaseError(err)
	}
	auth.ClearRefreshCookie(c, h.CookieCfg)
	return noContent(c)
}

func (h *AuthController) ForgotPassword(c echo.Context) error {
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

func (h *AuthController) ResetPassword(c echo.Context) error {
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

func (h *AuthController) Me(c echo.Context) error {
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

func (h *AuthController) ChangePassword(c echo.Context) error {
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

func (h *AuthController) CheckEmail(c echo.Context) error {
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

func (h *AuthController) GetProfile(c echo.Context) error {
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

func (h *AuthController) UpdateProfile(c echo.Context) error {
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
