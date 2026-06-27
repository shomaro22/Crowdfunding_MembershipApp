package controller

import (
	"context"

	"crowdfund-app/auth"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type AuthUseCase interface {
	RegisterBuyer(ctx context.Context, email, password string, name *string, meta usecase.RequestMeta) (usecase.UserView, error)
	RegisterSeller(ctx context.Context, in usecase.SellerRegIn, meta usecase.RequestMeta) (*usecase.SellerRegOut, error)
	Login(ctx context.Context, email, password string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	Refresh(ctx context.Context, refreshTokenPlain string, meta usecase.RequestMeta) (authToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshTokenPlain string, meta usecase.RequestMeta) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error
	GetUser(ctx context.Context, userID uuid.UUID) (usecase.UserView, error)
	EmailFree(ctx context.Context, email string) (bool, error)
	ParseToken(raw string) (auth.AuthClaims, error)
	Setup2FA(ctx context.Context, userID uuid.UUID) (*usecase.TOTPSetupResult, error)
	Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*usecase.TOTPVerifyResult, error)
	Challenge2FA(ctx context.Context, pendingToken, code string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	GoogleAuthURL(from string) (string, error)
	GoogleCallback(ctx context.Context, code, state string, meta usecase.RequestMeta) (*usecase.OAuthCBOut, error)
	ExchangeOAuthCode(ctx context.Context, code string, meta usecase.RequestMeta) (*usecase.LoginResult, error)
	SigningKey() []byte
}

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

type ProjectUseCase interface {
	Create(ctx context.Context, sellerID uuid.UUID, in usecase.CreateProjectIn) (*usecase.ProjectView, error)
	ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]usecase.ProjectView, int64, error)
	ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]usecase.ProjectView, int64, error)
	FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*usecase.ProjectView, error)
	ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]usecase.ProjectPlanView, error)
	Update(ctx context.Context, sellerID, projectID uuid.UUID, in usecase.UpdateProjectIn) (*usecase.ProjectView, error)
	RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*usecase.ProjectView, error)
}

type ConnectUseCase interface {
	GetAccount(ctx context.Context, sellerID uuid.UUID) (*usecase.ConnectAccountView, error)
	CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error)
}

type PaymentUseCase interface {
	CreateCheckout(ctx context.Context, in usecase.CheckoutIn) (usecase.CheckoutOut, error)
	CreateIntent(ctx context.Context, in usecase.IntentIn) (usecase.IntentOut, error)
	VerifyCheckout(ctx context.Context, buyerID uuid.UUID, sessionID string) (usecase.SupporterView, error)
	VerifyIntent(ctx context.Context, buyerID uuid.UUID, intentID string) (usecase.SupporterView, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

type PurchaseUseCase interface {
	ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]usecase.PurchaseView, int64, error)
}

type SupporterUseCase interface {
	ListMine(ctx context.Context, userID uuid.UUID) ([]usecase.SupporterView, error)
	Upgrade(ctx context.Context, userID, supporterID, planID uuid.UUID) (usecase.UpgradeOut, error)
	UpgradeConfirm(ctx context.Context, userID, supporterID uuid.UUID, intentID string) (usecase.SupporterView, error)
	Cancel(ctx context.Context, userID, supporterID uuid.UUID) (usecase.SupporterView, error)
	Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (usecase.SupporterView, error)
	ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in usecase.ChangeBillingIn) (usecase.SupporterView, error)
}

type MessageUseCase interface {
	ListThreads(ctx context.Context, userID uuid.UUID, q usecase.ListThreadsQuery) (usecase.ListThreadsResult, error)
	Send(ctx context.Context, senderID uuid.UUID, in usecase.SendMessageIn) (usecase.MessageItemView, error)
	MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta usecase.RequestMeta) error
	GetNotifications(ctx context.Context, userID uuid.UUID) (usecase.NotificationsView, error)
}

type MessageProjectAccess interface {
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
	ConfirmWSSubscribe(ctx context.Context, userID, projectID uuid.UUID, meta usecase.RequestMeta) error
}

type WSTokenUseCase interface {
	Issue(ctx context.Context, userID uuid.UUID) (string, error)
}
