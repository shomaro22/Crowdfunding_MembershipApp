package controller

import (
	"net/http"
	"strconv"
	"strings"

	"crowdfund-app/entity"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *AdminController) ListBuyerPending(c echo.Context) error {
	var (
		limit  int
		offset int
		users  []usecase.UserView
		total  int64
		items  []userJSON
		i      int
		err    error
	)
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	users, total, err = h.Admin.ListBuyerPending(c.Request().Context(), limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items = make([]userJSON, 0, len(users))
	for i = 0; i < len(users); i++ {
		items = append(items, userToJSON(users[i]))
	}
	return c.JSON(http.StatusOK, usersPageJSON{Users: items, Total: total})
}

func (h *AdminController) ApproveBuyer(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		u       usecase.UserView
		err     error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err = uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	u, err = h.Admin.ApproveBuyer(c.Request().Context(), adminID, userID, nil)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) RejectBuyer(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		body    struct {
			Reason *string `json:"reason"`
		}
		u   usecase.UserView
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
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	if body.Reason == nil || strings.TrimSpace(*body.Reason) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	u, err = h.Admin.RejectBuyer(c.Request().Context(), adminID, userID, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) ListSellerRegs(c echo.Context) error {
	var (
		status entity.RegStatus
		limit  int
		offset int
		regs   []usecase.SellerRegItemView
		total  int64
		items  []sellerRegJSON
		i      int
		err    error
	)
	status = entity.RegStatus(c.QueryParam("status"))
	if status == "" {
		status = entity.RegPending
	}
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	regs, total, err = h.Admin.ListSellerRegs(c.Request().Context(), status, limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items = make([]sellerRegJSON, 0, len(regs))
	for i = 0; i < len(regs); i++ {
		items = append(items, sellerRegToJSON(regs[i]))
	}
	return c.JSON(http.StatusOK, sellerRegsPageJSON{Registrations: items, Total: total})
}

func (h *AdminController) GetSellerReg(c echo.Context) error {
	var (
		id     uuid.UUID
		detail *usecase.SellerRegView
		err    error
	)
	id, err = uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	detail, err = h.Admin.GetSellerReg(c.Request().Context(), id)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegViewToJSON(detail))
}

func (h *AdminController) ApproveSellerReg(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		reg     usecase.SellerRegItemView
		err     error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	reg, err = h.Admin.ApproveSellerReg(c.Request().Context(), adminID, id)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}

func (h *AdminController) RejectSellerReg(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		body    struct {
			Reason string `json:"reason"`
		}
		reg usecase.SellerRegItemView
		err error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err = uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil || body.Reason == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	reg, err = h.Admin.RejectSellerReg(c.Request().Context(), adminID, id, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}
