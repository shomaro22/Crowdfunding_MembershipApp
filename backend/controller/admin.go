package controller

import (
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminController struct {
	Admin        usecase.IAdminUsecase
	APIPublicURL string
}

func NewAdminController(admin usecase.IAdminUsecase, apiPublicURL string) *AdminController {
	return &AdminController{Admin: admin, APIPublicURL: apiPublicURL}
}

func (h *AdminController) ResetUser2FA(c echo.Context) error {
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
	_ = c.Bind(&body)
	u, err = h.Admin.ResetUser2FA(reqCtx(c), adminID, userID, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]UserWire{"user": UserWireFrom(u)})
}

func (h *AdminController) ChangeAdminRole(c echo.Context) error {
	var (
		actorID uuid.UUID
		userID  uuid.UUID
		body    struct {
			AdminRole string  `json:"admin_role"`
			Reason    *string `json:"reason"`
		}
		u   usecase.UserView
		err error
	)
	actorID, err = requireUserID(c)
	if err != nil {
		return err
	}
	userID, err = pathUUID(c, "userId", "invalid user id")
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil || body.AdminRole == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("admin_role required", "BAD_REQUEST"))
	}
	u, err = h.Admin.ChangeAdminRole(reqCtx(c), actorID, userID, body.AdminRole, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]UserWire{"user": UserWireFrom(u)})
}

func (h *AdminController) DashboardStats(c echo.Context) error {
	var (
		stats usecase.AdminStatsView
		items []UserWire
		i     int
		err   error
	)
	stats, err = h.Admin.DashboardStats(reqCtx(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]UserWire, 0, len(stats.RecentUsers))
	for i = range stats.RecentUsers {
		items = append(items, UserWireFrom(stats.RecentUsers[i]))
	}
	return jsonOK(c, adminStatsWire{
		TotalUsers:     stats.TotalUsers,
		TotalProjects:  stats.TotalProjects,
		TotalPurchases: stats.TotalPurchases,
		TotalRevenue:   stats.TotalRevenue,
		RecentUsers:    items,
	})
}

func (h *AdminController) ListUsers(c echo.Context) error {
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
	users, total, err = h.Admin.ListUsers(reqCtx(c), c.QueryParam("role"), c.QueryParam("search"), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]UserWire, 0, len(users))
	for i = range users {
		items = append(items, UserWireFrom(users[i]))
	}
	return jsonOK(c, usersPageJSON{Users: items, Total: total})
}

func (h *AdminController) UpdateUser(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		body    adminUpdateUserJSON
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
	if err = bindBody(c, &body); err != nil {
		return err
	}
	u, err = h.Admin.UpdateUser(reqCtx(c), adminID, userID, usecase.AdminUpdateUserIn{
		Email:    body.Email,
		Name:     body.Name,
		Role:     body.Role,
		IsActive: body.IsActive,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, UserWireFrom(u))
}

func (h *AdminController) DeleteUser(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
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
	if err = h.Admin.DeleteUser(reqCtx(c), adminID, userID); err != nil {
		return mapUseCaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminController) ListPurchases(c echo.Context) error {
	var (
		limit   int
		offset  int
		status  string
		statusQ *string
		rows    []usecase.PurchaseView
		total   int64
		err     error
	)
	limit, offset = queryPage(c)
	statusQ = queryOptionalString(c, "status")
	if statusQ != nil {
		status = *statusQ
	}
	rows, total, err = h.Admin.ListPurchases(reqCtx(c), status, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listPurchasesJSON(rows, total))
}

func (h *AdminController) PatchPurchase(c echo.Context) error {
	var (
		adminID    uuid.UUID
		purchaseID uuid.UUID
		body       adminPatchPurchaseJSON
		view       usecase.PurchaseView
		err        error
	)
	adminID, err = requireUserID(c)
	if err != nil {
		return err
	}
	purchaseID, err = pathUUID(c, "purchaseId", "invalid purchase id")
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil || body.Status == "" {
		return badRequest(c, "status required")
	}
	view, err = h.Admin.PatchPurchaseStatus(reqCtx(c), adminID, purchaseID, body.Status)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, purchaseToJSON(view))
}
