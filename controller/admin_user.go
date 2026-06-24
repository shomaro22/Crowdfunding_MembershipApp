package controller

import (
	"net/http"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *AdminController) ResetUser2FA(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		body    struct {
			Reason *string `json:"reason"`
		}
		u   *entity.User
		err error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err = uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	_ = c.Bind(&body)
	u, err = h.Admin.ResetUser2FA(c.Request().Context(), adminID, userID, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, map[string]userJSON{"user": userToJSON(u)})
}

func (h *AdminController) ChangeAdminRole(c echo.Context) error {
	var (
		actorID uuid.UUID
		userID  uuid.UUID
		body    struct {
			AdminRole string  `json:"admin_role"`
			Reason    *string `json:"reason"`
		}
		u   *entity.User
		err error
	)
	actorID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err = uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil || body.AdminRole == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("admin_role required", "BAD_REQUEST"))
	}
	u, err = h.Admin.ChangeAdminRole(c.Request().Context(), actorID, userID, entity.AdminRole(body.AdminRole), body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, map[string]userJSON{"user": userToJSON(u)})
}
