package controller

import (
	"net/http"
	"strings"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *AdminController) ListProjects(c echo.Context) error {
	var (
		status *string
		s      string
		limit  int
		offset int
		rows   []usecase.ProjectView
		total  int64
		err    error
	)
	s = c.QueryParam("status")
	if s != "" {
		status = &s
	}
	limit, offset = queryPage(c)
	rows, total, err = h.Admin.ListProjects(reqCtx(c), status, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listProjectsJSON(rows, total, limit, offset))
}

func (h *AdminController) PatchProject(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		body    struct {
			Status string  `json:"status"`
			Reason *string `json:"reason"`
		}
		view *usecase.ProjectView
		err  error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("status required", "BAD_REQUEST"))
	}
	view, err = h.Admin.PatchProject(reqCtx(c), adminID, id, body.Status, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectDetailJSON(view))
}

func (h *AdminController) DeleteProject(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		err     error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	if err = h.Admin.DeleteProject(reqCtx(c), adminID, id); err != nil {
		return mapUseCaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminController) PatchDeleteReq(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		body    struct {
			Action string  `json:"action"`
			Reason *string `json:"reason"`
		}
		view *usecase.ProjectView
		err  error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "projectId", "invalid project id")
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.Action == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("action required", "BAD_REQUEST"))
	}
	view, err = h.Admin.PatchDeleteReq(reqCtx(c), adminID, id, body.Action, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, projectDetailJSON(view))
}

func (h *AdminController) ListBuyerPending(c echo.Context) error {
	var (
		limit  int
		offset int
		users  []usecase.UserView
		total  int64
		items  []UserWire
		i      int
		err    error
	)
	limit, offset = queryPage(c)
	users, total, err = h.Admin.ListBuyerPending(reqCtx(c), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]UserWire, 0, len(users))
	for i = 0; i < len(users); i++ {
		items = append(items, UserWireFrom(users[i]))
	}
	return jsonOK(c, usersPageJSON{Users: items, Total: total})
}

func (h *AdminController) ApproveBuyer(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		u       usecase.UserView
		err     error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	userID, err = pathUUID(c, "userId", "invalid user id")
	if err != nil {
		return err
	}
	u, err = h.Admin.ApproveBuyer(reqCtx(c), adminID, userID, nil)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, UserWireFrom(u))
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
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	userID, err = pathUUID(c, "userId", "invalid user id")
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	if body.Reason == nil || strings.TrimSpace(*body.Reason) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	u, err = h.Admin.RejectBuyer(reqCtx(c), adminID, userID, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, UserWireFrom(u))
}

func (h *AdminController) ListSellerRegs(c echo.Context) error {
	var (
		status string
		limit  int
		offset int
		regs   []usecase.SellerRegItemView
		total  int64
		items  []sellerRegJSON
		i      int
		err    error
	)
	status = c.QueryParam("status")
	limit, offset = queryPage(c)
	regs, total, err = h.Admin.ListSellerRegs(reqCtx(c), status, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]sellerRegJSON, 0, len(regs))
	for i = 0; i < len(regs); i++ {
		items = append(items, sellerRegToJSON(regs[i]))
	}
	return jsonOK(c, sellerRegsPageJSON{Registrations: items, Total: total})
}

func (h *AdminController) GetSellerReg(c echo.Context) error {
	var (
		id     uuid.UUID
		detail *usecase.SellerRegView
		err    error
	)
	id, err = pathUUID(c, "registrationId", "invalid registration id")
	if err != nil {
		return err
	}
	detail, err = h.Admin.GetSellerReg(reqCtx(c), id)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, sellerRegViewToJSON(h.APIPublicURL, detail))
}

func (h *AdminController) ApproveSellerReg(c echo.Context) error {
	var (
		adminID uuid.UUID
		id      uuid.UUID
		reg     usecase.SellerRegItemView
		err     error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "registrationId", "invalid registration id")
	if err != nil {
		return err
	}
	reg, err = h.Admin.ApproveSellerReg(reqCtx(c), adminID, id)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, sellerRegToJSON(reg))
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
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	id, err = pathUUID(c, "registrationId", "invalid registration id")
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.Reason == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	reg, err = h.Admin.RejectSellerReg(reqCtx(c), adminID, id, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, sellerRegToJSON(reg))
}
