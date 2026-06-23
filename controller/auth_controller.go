package controller

import (
	"net/http"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthController struct {
	Auth       AuthUseCase
	CookieCfg  auth.CookieCfg
}

func NewAuthController(authUC AuthUseCase, cookieCfg auth.CookieCfg) *AuthController {
	return &AuthController{Auth: authUC, CookieCfg: cookieCfg}
}

type registerRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
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

func (h *AuthController) Register(c echo.Context) error {
	var (
		body registerRequest
		u    *entity.User
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	u, err = h.Auth.RegisterBuyer(c.Request().Context(), body.Email, body.Password, body.Name, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusCreated, registerResponse{
		User:    userToJSON(u),
		Message: "Registration pending admin approval",
	})
}

func (h *AuthController) Login(c echo.Context) error {
	var (
		body loginRequest
		res  *usecase.LoginResult
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	res, err = h.Auth.Login(c.Request().Context(), body.Email, body.Password, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	if res.Requires2FA {
		auth.SetPending2FA(c, h.CookieCfg, h.Auth.SigningKey(), res.User.ID)
		return c.JSON(http.StatusOK, login2FAResponse{
			Requires2FA: true,
			User:        userToJSON(res.User),
		})
	}
	auth.SetReCookie(c, h.CookieCfg, res.ReToken)
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      userToJSON(res.User),
	})
}

func (h *AuthController) Re(c echo.Context) error {
	var (
		plain     string
		authToken string
		reToken   string
		err       error
	)
	plain = auth.ReTokenFrom(c, h.CookieCfg)
	authToken, reToken, err = h.Auth.Reissue(c.Request().Context(), plain, authMeta(c))
	if err != nil {
		return mapAuthError(err)
	}
	auth.SetReCookie(c, h.CookieCfg, reToken)
	return c.JSON(http.StatusOK, map[string]string{"auth_token": authToken})
}

func (h *AuthController) Logout(c echo.Context) error {
	var (
		plain string
		err   error
	)
	plain = auth.ReTokenFrom(c, h.CookieCfg)
	if err = h.Auth.Logout(c.Request().Context(), plain, authMeta(c)); err != nil {
		return mapAuthError(err)
	}
	auth.ClearReCookie(c, h.CookieCfg)
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) ForgotPassword(c echo.Context) error {
	var (
		body forgotPasswordRequest
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	_ = h.Auth.ForgotPassword(c.Request().Context(), body.Email)
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) ResetPassword(c echo.Context) error {
	var (
		body resetPasswordRequest
		err  error
	)
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	if err = h.Auth.ResetPassword(c.Request().Context(), body.Token, body.Password); err != nil {
		return mapAuthError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) Me(c echo.Context) error {
	var (
		uid uuid.UUID
		u   *entity.User
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	u, err = h.Auth.GetUser(c.Request().Context(), uid)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]userJSON{"user": userToJSON(u)})
}

func (h *AuthController) ChangePassword(c echo.Context) error {
	var (
		uid  uuid.UUID
		body changePwRequest
		err  error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	if err = h.Auth.ChangePassword(c.Request().Context(), uid, body.CurrentPassword, body.NewPassword, body.TotpCode); err != nil {
		return mapAuthError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) CheckEmail(c echo.Context) error {
	var (
		email     string
		available bool
		err       error
	)
	email = c.QueryParam("email")
	available, err = h.Auth.EmailFree(c.Request().Context(), email)
	if err != nil {
		return mapAuthError(err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"available": available})
}
