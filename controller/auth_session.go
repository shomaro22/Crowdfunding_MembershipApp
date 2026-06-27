package controller

import (
	"net/http"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

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
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	res, err = h.Auth.Login(c.Request().Context(), body.Email, body.Password, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	if res.Requires2FA {
		auth.SetPending2FA(c, h.CookieCfg, h.Auth.SigningKey(), res.User.ID)
		return c.JSON(http.StatusOK, login2FAResponse{
			Requires2FA: true,
			User:        usecase.UserWireFrom(res.User),
		})
	}
	auth.SetRefreshCookie(c, h.CookieCfg, res.RefreshToken)
	return c.JSON(http.StatusOK, authTokenResponse{
		AuthToken: res.AuthToken,
		User:      usecase.UserWireFrom(res.User),
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
	authToken, refreshToken, err = h.Auth.Refresh(c.Request().Context(), plain, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	auth.SetRefreshCookie(c, h.CookieCfg, refreshToken)
	return c.JSON(http.StatusOK, map[string]string{"auth_token": authToken})
}

func (h *AuthController) Logout(c echo.Context) error {
	var (
		plain string
		err   error
	)
	plain = auth.RefreshTokenFrom(c, h.CookieCfg)
	if err = h.Auth.Logout(c.Request().Context(), plain, requestMeta(c)); err != nil {
		return mapUseCaseError(err)
	}
	auth.ClearRefreshCookie(c, h.CookieCfg)
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
		return mapUseCaseError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *AuthController) Me(c echo.Context) error {
	var (
		uid uuid.UUID
		u   usecase.UserView
		err error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	u, err = h.Auth.GetUser(c.Request().Context(), uid)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]usecase.UserWire{"user": usecase.UserWireFrom(u)})
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
		return mapUseCaseError(err)
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
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"available": available})
}
