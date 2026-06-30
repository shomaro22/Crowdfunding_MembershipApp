package usecase

import (
	"context"
	"os"
	"time"

	"crowdfund-app/model"

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

type IProjectUsecase interface {
	Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error)
	ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]ProjectView, int64, error)
	ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]ProjectView, int64, error)
	FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*ProjectView, error)
	ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]ProjectPlanView, error)
	Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error)
	RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error)
	EnrichList(ctx context.Context, rows []model.Project, total int64) ([]ProjectView, int64, error)
}

type IConnectUsecase interface {
	GetAccount(ctx context.Context, sellerID uuid.UUID) (*ConnectAccountView, error)
	CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error)
}

type IPaymentUsecase interface {
	CreateCheckout(ctx context.Context, in CheckoutIn) (CheckoutOut, error)
	CreateIntent(ctx context.Context, in IntentIn) (IntentOut, error)
	VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (SupporterView, error)
	VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (SupporterView, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

type IPurchaseUsecase interface {
	ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PurchaseView, int64, error)
}

type ISupporterUsecase interface {
	ListMine(ctx context.Context, userID uuid.UUID) ([]SupporterView, error)
	Upgrade(ctx context.Context, userID, supporterID, planID uuid.UUID) (UpgradeOut, error)
	UpgradeConfirm(ctx context.Context, userID, supporterID uuid.UUID, intentID string) (SupporterView, error)
	Cancel(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error)
	Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (SupporterView, error)
	ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in ChangeBillingIn) (SupporterView, error)
}

type IMessageUsecase interface {
	ListThreads(ctx context.Context, userID uuid.UUID, q ListThreadsQuery) (ListThreadsResult, error)
	Send(ctx context.Context, senderID uuid.UUID, in SendMessageIn) (MessageItemView, error)
	MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta RequestMeta) error
	GetNotifications(ctx context.Context, userID uuid.UUID) (NotificationsView, error)
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
	ConfirmWSSubscribe(ctx context.Context, userID, projectID uuid.UUID, meta RequestMeta) error
}

type IWSTokenUsecase interface {
	Issue(ctx context.Context, userID uuid.UUID) (string, error)
	Consume(ctx context.Context, plain string, meta RequestMeta) (uuid.UUID, error)
}

type IFavoriteUsecase interface {
	List(ctx context.Context, userID uuid.UUID) ([]ProjectView, error)
	Toggle(ctx context.Context, userID uuid.UUID, in ToggleFavoriteIn) (FavoriteStatusView, error)
	Check(ctx context.Context, userID, projectID uuid.UUID) (FavoriteStatusView, error)
}

type IReportUsecase interface {
	Create(ctx context.Context, reporterID uuid.UUID, in CreateReportIn) (ReportView, error)
	ListAdmin(ctx context.Context, status *string, limit, offset int) (ReportsResult, error)
	UpdateAdmin(ctx context.Context, adminID, reportID uuid.UUID, in UpdateReportIn) (ReportView, error)
}

type IMediaUsecase interface {
	UploadImage(ctx context.Context, userID uuid.UUID, in MediaUploadIn) (string, error)
}

type ISellerUsecase interface {
	DashboardStats(ctx context.Context, sellerID uuid.UUID) (SellerDashboardStatsView, error)
	ListSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID, limit, offset int) (SellerSupportersResult, error)
	ExportSupporters(ctx context.Context, sellerID uuid.UUID, projectID *uuid.UUID) ([]SupporterView, error)
	ProjectAnalysis(ctx context.Context, sellerID uuid.UUID) (ProjectAnalysisResult, error)
}

type IOperationsUsecase interface {
	ListActivityLogs(ctx context.Context, q ActivityLogsQuery) (ActivityLogsResult, error)
	ListAuditLogs(ctx context.Context, q AuditLogsQuery) (AuditLogsResult, error)
	ListBatchRuns(ctx context.Context, q BatchRunsQuery) (BatchRunsResult, error)
	GetSecurityEvents(ctx context.Context, from, to *time.Time) (SecurityEventsView, error)
	RunBatch(ctx context.Context, adminID uuid.UUID, jobName string, dryRun bool) (*BatchRunView, error)
}

type IKYCAccessUsecase interface {
	OpenWithToken(ctx context.Context, token string) (*os.File, string, error)
	SignToken(storageKey string, now time.Time) string
}

type IRateLimitUsecase interface {
	Allow(ctx context.Context, key string, limit int, period time.Duration) (bool, int, error)
}
