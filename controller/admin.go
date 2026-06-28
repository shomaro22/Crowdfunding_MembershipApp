package controller

import (
	"net/http"
	"strconv"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminController struct {
	Admin        AdminUseCase
	APIPublicURL string
}

func NewAdminController(admin AdminUseCase, apiPublicURL string) *AdminController {
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
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]UserWire{"user": UserWireFrom(u)})
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
	u, err = h.Admin.ChangeAdminRole(c.Request().Context(), actorID, userID, body.AdminRole, body.Reason)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]UserWire{"user": UserWireFrom(u)})
}

func (h *AdminController) DashboardStats(c echo.Context) error {
	var (
		stats usecase.AdminStatsView
		items []UserWire
		i     int
		err   error
	)
	stats, err = h.Admin.DashboardStats(c.Request().Context())
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]UserWire, 0, len(stats.RecentUsers))
	for i = range stats.RecentUsers {
		items = append(items, UserWireFrom(stats.RecentUsers[i]))
	}
	return c.JSON(http.StatusOK, adminStatsWire{
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
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	users, total, err = h.Admin.ListUsers(c.Request().Context(), c.QueryParam("role"), c.QueryParam("search"), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	items = make([]UserWire, 0, len(users))
	for i = range users {
		items = append(items, UserWireFrom(users[i]))
	}
	return c.JSON(http.StatusOK, usersPageJSON{Users: items, Total: total})
}

func (h *AdminController) UpdateUser(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
		body    adminUpdateUserJSON
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
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	u, err = h.Admin.UpdateUser(c.Request().Context(), adminID, userID, usecase.AdminUpdateUserIn{
		Email:    body.Email,
		Name:     body.Name,
		Role:     body.Role,
		IsActive: body.IsActive,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, UserWireFrom(u))
}

func (h *AdminController) DeleteUser(c echo.Context) error {
	var (
		adminID uuid.UUID
		userID  uuid.UUID
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
	if err = h.Admin.DeleteUser(c.Request().Context(), adminID, userID); err != nil {
		return mapUseCaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminController) ListPurchases(c echo.Context) error {
	var (
		limit  int
		offset int
		rows   []usecase.PurchaseView
		total  int64
		err    error
	)
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	rows, total, err = h.Admin.ListPurchases(c.Request().Context(), c.QueryParam("status"), limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listPurchasesJSON(rows, total))
}

func (h *AdminController) PatchPurchase(c echo.Context) error {
	var (
		adminID    uuid.UUID
		purchaseID uuid.UUID
		body       adminPatchPurchaseJSON
		view       usecase.PurchaseView
		err        error
	)
	adminID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	purchaseID, err = uuid.Parse(c.Param("purchaseId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid purchase id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil || body.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("status required", "BAD_REQUEST"))
	}
	view, err = h.Admin.PatchPurchaseStatus(c.Request().Context(), adminID, purchaseID, body.Status)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, purchaseToJSON(view))
}
