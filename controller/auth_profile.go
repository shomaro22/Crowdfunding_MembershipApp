package controller

import (
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type updateProfileJSON struct {
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

func (h *AuthController) GetProfile(c echo.Context) error {
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
	return c.JSON(http.StatusOK, UserWireFrom(u))
}

func (h *AuthController) UpdateProfile(c echo.Context) error {
	var (
		uid  uuid.UUID
		body updateProfileJSON
		u    usecase.UserView
		err  error
	)
	uid, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	u, err = h.Auth.UpdateProfile(c.Request().Context(), uid, usecase.UpdateProfileIn{
		Name:      body.Name,
		AvatarURL: body.AvatarURL,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, UserWireFrom(u))
}
