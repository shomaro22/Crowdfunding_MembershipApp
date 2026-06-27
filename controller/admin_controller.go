package controller

import (
	"context"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminUseCase interface {
	ListBuyerPending(ctx context.Context, limit, offset int) ([]usecase.UserView, int64, error)
	ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (usecase.UserView, error)
	RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (usecase.UserView, error)
	ListSellerRegs(ctx context.Context, status string, limit, offset int) ([]usecase.SellerRegItemView, int64, error)
	GetSellerReg(ctx context.Context, id uuid.UUID) (*usecase.SellerRegView, error)
	ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (usecase.SellerRegItemView, error)
	RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (usecase.SellerRegItemView, error)
	ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (usecase.UserView, error)
	ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole string, reason *string) (usecase.UserView, error)
	ListProjects(ctx context.Context, status *string, limit, offset int) ([]usecase.ProjectView, int64, error)
	PatchProject(ctx context.Context, adminID, projectID uuid.UUID, status string, reason *string) (*usecase.ProjectView, error)
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
	return mapUseCaseError(err)
}
