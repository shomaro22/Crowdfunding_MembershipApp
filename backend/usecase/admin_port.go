package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IAdminUsecase interface {
	ListBuyerPending(ctx context.Context, limit, offset int) ([]UserView, int64, error)
	ApproveBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error)
	RejectBuyer(ctx context.Context, adminID, userID uuid.UUID, reason *string) (UserView, error)
	ListSellerRegs(ctx context.Context, status string, limit, offset int) ([]SellerRegItemView, int64, error)
	GetSellerReg(ctx context.Context, id uuid.UUID) (*SellerRegView, error)
	ApproveSellerReg(ctx context.Context, adminID, regID uuid.UUID) (SellerRegItemView, error)
	RejectSellerReg(ctx context.Context, adminID, regID uuid.UUID, reason string) (SellerRegItemView, error)
	ResetUser2FA(ctx context.Context, adminID, targetUserID uuid.UUID, reason *string) (UserView, error)
	ChangeAdminRole(ctx context.Context, actorID, targetUserID uuid.UUID, newRole string, reason *string) (UserView, error)
	ListProjects(ctx context.Context, status *string, limit, offset int) ([]ProjectView, int64, error)
	PatchProject(ctx context.Context, adminID, projectID uuid.UUID, status string, reason *string) (*ProjectView, error)
	DeleteProject(ctx context.Context, adminID, projectID uuid.UUID) error
	PatchDeleteReq(ctx context.Context, adminID, projectID uuid.UUID, action string, reason *string) (*ProjectView, error)
	DashboardStats(ctx context.Context) (AdminStatsView, error)
	ListUsers(ctx context.Context, role, search string, limit, offset int) ([]UserView, int64, error)
	UpdateUser(ctx context.Context, adminID, userID uuid.UUID, in AdminUpdateUserIn) (UserView, error)
	DeleteUser(ctx context.Context, adminID, userID uuid.UUID) error
	ListPurchases(ctx context.Context, status string, limit, offset int) ([]PurchaseView, int64, error)
	PatchPurchaseStatus(ctx context.Context, adminID, purchaseID uuid.UUID, status string) (PurchaseView, error)
}

type IOperationsUsecase interface {
	ListActivityLogs(ctx context.Context, q ActivityLogsQuery) (ActivityLogsResult, error)
	ListAuditLogs(ctx context.Context, q AuditLogsQuery) (AuditLogsResult, error)
	ListBatchRuns(ctx context.Context, q BatchRunsQuery) (BatchRunsResult, error)
	GetSecurityEvents(ctx context.Context, from, to *time.Time) (SecurityEventsView, error)
	RunBatch(ctx context.Context, adminID uuid.UUID, jobName string, dryRun bool) (*BatchRunView, error)
}
