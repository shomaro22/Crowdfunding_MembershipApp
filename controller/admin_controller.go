package controller

import (
	"context"
	"errors"
	"net/http"

	"crowdfund-app/entity"
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
	ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (*entity.User, error)
	ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole entity.AdminRole, reason *string) (*entity.User, error)
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
