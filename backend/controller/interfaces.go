package controller

import "github.com/labstack/echo/v4"


type IHealthController interface {
	Health(c echo.Context) error
}

type IAuthController interface {
	Register(c echo.Context) error
	RegisterSeller(c echo.Context) error
	Login(c echo.Context) error
	Logout(c echo.Context) error
	LogoutAllDevices(c echo.Context) error
	ForgotPassword(c echo.Context) error
	ResetPassword(c echo.Context) error
	CheckEmail(c echo.Context) error
	Refresh(c echo.Context) error
	Challenge2FA(c echo.Context) error
	GoogleStart(c echo.Context) error
	GoogleCallback(c echo.Context) error
	OAuthExchange(c echo.Context) error
	OAuthLinkConfirm(c echo.Context) error
	Me(c echo.Context) error
	GetProfile(c echo.Context) error
	UpdateProfile(c echo.Context) error
	ChangePassword(c echo.Context) error
	Setup2FA(c echo.Context) error
	Verify2FA(c echo.Context) error
}

type IProjectController interface {
	ListPublic(c echo.Context) error
	ListFeatured(c echo.Context) error
	ListMine(c echo.Context) error
	Create(c echo.Context) error
	Get(c echo.Context) error
	ListPlans(c echo.Context) error
	Update(c echo.Context) error
	RequestDelete(c echo.Context) error
}

type IConnectController interface {
	GetAccount(c echo.Context) error
	CreateOnboarding(c echo.Context) error
}

type ISellerController interface {
	DashboardStats(c echo.Context) error
	ListSupporters(c echo.Context) error
	ExportSupporters(c echo.Context) error
	ProjectAnalysis(c echo.Context) error
}

type IPaymentController interface {
	Checkout(c echo.Context) error
	Intent(c echo.Context) error
	VerifyCheckout(c echo.Context) error
	VerifyIntent(c echo.Context) error
	Webhook(c echo.Context) error
	ListPurchases(c echo.Context) error
}

type ISupporterController interface {
	List(c echo.Context) error
	Cancel(c echo.Context) error
	Rejoin(c echo.Context) error
	ChangeBilling(c echo.Context) error
}

type IFavoriteController interface {
	List(c echo.Context) error
	Toggle(c echo.Context) error
	Check(c echo.Context) error
}

type IReportController interface {
	Create(c echo.Context) error
	ListAdmin(c echo.Context) error
	PatchAdmin(c echo.Context) error
}

type IMediaController interface {
	UploadImage(c echo.Context) error
}

type IMessageController interface {
	List(c echo.Context) error
	Send(c echo.Context) error
	MarkRead(c echo.Context) error
	Notifications(c echo.Context) error
}

type IWSTokenController interface {
	Issue(c echo.Context) error
}

type IWSController interface {
	Handle(c echo.Context) error
}

type IAdminController interface {
	DashboardStats(c echo.Context) error
	ListUsers(c echo.Context) error
	UpdateUser(c echo.Context) error
	DeleteUser(c echo.Context) error
	ListPurchases(c echo.Context) error
	PatchPurchase(c echo.Context) error
	ListBuyerPending(c echo.Context) error
	ApproveBuyer(c echo.Context) error
	RejectBuyer(c echo.Context) error
	ListSellerRegs(c echo.Context) error
	GetSellerReg(c echo.Context) error
	ApproveSellerReg(c echo.Context) error
	RejectSellerReg(c echo.Context) error
	ListProjects(c echo.Context) error
	PatchProject(c echo.Context) error
	DeleteProject(c echo.Context) error
	PatchDeleteReq(c echo.Context) error
	ResetUser2FA(c echo.Context) error
	ChangeAdminRole(c echo.Context) error
}

type IAdminOperationsController interface {
	ListActivity(c echo.Context) error
	ListAuditLogs(c echo.Context) error
	ListBatches(c echo.Context) error
	RunBatch(c echo.Context) error
	GetSecurity(c echo.Context) error
}

type IKYCFileController interface {
	Serve(c echo.Context) error
}
