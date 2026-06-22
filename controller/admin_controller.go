package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/storage"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminUseCase interface {
	ListBuyerPending(ctx context.Context, limit, offset int) ([]entity.User, int64, error)
	ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error)
	RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (*entity.User, error)
	ListSellerRegs(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error)
	GetSellerReg(ctx context.Context, id uuid.UUID) (*usecase.SellerRegView, error)
	ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (*entity.SellerReg, error)
	RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (*entity.SellerReg, error)
	ListProjects(ctx context.Context, status *entity.ProjectStatus, limit, offset int) ([]usecase.ProjectView, int64, error)
	PatchProject(ctx context.Context, adminID, projectID uuid.UUID, status entity.ProjectStatus, reason *string) (*usecase.ProjectView, error)
	DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error
	PatchDeleteReq(ctx context.Context, adminID, projectID uuid.UUID, action string, reason *string) (*usecase.ProjectView, error)
}

type AdminController struct {
	Admin AdminUseCase
}

func NewAdminController(admin AdminUseCase) *AdminController {
	return &AdminController{Admin: admin}
}

func (h *AdminController) ListBuyerPending(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	users, total, err := h.Admin.ListBuyerPending(c.Request().Context(), limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items := make([]userJSON, 0, len(users))
	for i := range users {
		items = append(items, userToJSON(&users[i]))
	}
	return c.JSON(http.StatusOK, usersPageJSON{Users: items, Total: total})
}

func (h *AdminController) ApproveBuyer(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	u, err := h.Admin.ApproveBuyer(c.Request().Context(), adminID, userID, nil)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) RejectBuyer(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid user id", "BAD_REQUEST"))
	}
	var body struct {
		Reason *string `json:"reason"`
	}
	_ = c.Bind(&body)
	u, err := h.Admin.RejectBuyer(c.Request().Context(), adminID, userID, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, userToJSON(u))
}

func (h *AdminController) ListSellerRegs(c echo.Context) error {
	status := entity.RegStatus(c.QueryParam("status"))
	if status == "" {
		status = entity.RegPending
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	regs, total, err := h.Admin.ListSellerRegs(c.Request().Context(), status, limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	items := make([]sellerRegJSON, 0, len(regs))
	for i := range regs {
		items = append(items, sellerRegToJSON(&regs[i]))
	}
	return c.JSON(http.StatusOK, sellerRegsPageJSON{Registrations: items, Total: total})
}

func (h *AdminController) GetSellerReg(c echo.Context) error {
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	detail, err := h.Admin.GetSellerReg(c.Request().Context(), id)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegViewToJSON(detail))
}

func (h *AdminController) ApproveSellerReg(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	reg, err := h.Admin.ApproveSellerReg(c.Request().Context(), adminID, id)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}

func (h *AdminController) RejectSellerReg(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("registrationId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid registration id", "BAD_REQUEST"))
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil || body.Reason == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("reason required", "BAD_REQUEST"))
	}
	reg, err := h.Admin.RejectSellerReg(c.Request().Context(), adminID, id, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, sellerRegToJSON(reg))
}

func (h *AdminController) ListProjects(c echo.Context) error {
	var status *entity.ProjectStatus
	if s := c.QueryParam("status"); s != "" {
		st := entity.ProjectStatus(s)
		status = &st
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	rows, total, err := h.Admin.ListProjects(c.Request().Context(), status, limit, offset)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, listProjectsJSON(rows, total, limit, offset))
}

func (h *AdminController) PatchProject(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	var body struct {
		Status string  `json:"status"`
		Reason *string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil || body.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("status required", "BAD_REQUEST"))
	}
	view, err := h.Admin.PatchProject(c.Request().Context(), adminID, id, entity.ProjectStatus(body.Status), body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

func (h *AdminController) DeleteProject(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	if err := h.Admin.DeleteProject(c.Request().Context(), adminID, id); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminController) PatchDeleteReq(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid project id", "BAD_REQUEST"))
	}
	var body struct {
		Action string  `json:"action"`
		Reason *string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil || body.Action == "" {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("action required", "BAD_REQUEST"))
	}
	view, err := h.Admin.PatchDeleteReq(c.Request().Context(), adminID, id, body.Action, body.Reason)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, projectDetailJSON(view))
}

type KYCFileController struct {
	KYC        storage.KYCStore
	SignSecret []byte
}

func NewKYCFileController(kyc storage.KYCStore, secret []byte) *KYCFileController {
	return &KYCFileController{KYC: kyc, SignSecret: secret}
}

func (h *KYCFileController) Serve(c echo.Context) error {
	token := c.QueryParam("token")
	key, err := storage.VerifyKYCAccess(h.SignSecret, token, time.Now().UTC())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid or expired token", "INVALID_TOKEN"))
	}
	f, ct, err := h.KYC.Open(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	}
	defer f.Close()
	return c.Stream(http.StatusOK, ct, f)
}

func mapAdminError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrInvalidState):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid state", "INVALID_STATE"))
	case errors.Is(err, usecase.ErrBadRequest):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	case errors.Is(err, usecase.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, apiError("forbidden", "FORBIDDEN"))
	default:
		return mapAuthError(err)
	}
}
