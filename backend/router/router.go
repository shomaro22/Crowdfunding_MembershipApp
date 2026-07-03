package router

import (
	"os"

	"crowdfund-app/controller"
	"crowdfund-app/model"
	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type Config struct {
	Health             controller.IHealthController
	AuthUsecase        usecase.IAuthUsecase
	AppPublicURL       string
	Projects           controller.IProjectController
	Connect            controller.IConnectController
	Seller             controller.ISellerController
	Payments           controller.IPaymentController
	Supporters         controller.ISupporterController
	Messages           controller.IMessageController
	WSToken            controller.IWSTokenController
	WS                 controller.IWSController
	Favorites          controller.IFavoriteController
	Reports            controller.IReportController
	Media              controller.IMediaController
	Admin              controller.IAdminController
	AdminOperations    controller.IAdminOperationsController
	KYCFiles           controller.IKYCFileController
	RateLimit         *middleware.RateLimiter
	Policy2FAChallenge middleware.RatePolicy
	OAuthEnabled      bool
	WSTokenAuth       echo.MiddlewareFunc
	PreAuthMiddleware []echo.MiddlewareFunc
	Protected         []echo.MiddlewareFunc
	OptionalJWT       []echo.MiddlewareFunc
	SellerRead        []echo.MiddlewareFunc
	SellerWrite       []echo.MiddlewareFunc
	AdminOnly         []echo.MiddlewareFunc
}

func New(cfg Config) *echo.Echo {
	var (
		e       *echo.Echo
		mw      echo.MiddlewareFunc
		rl      func(policy middleware.RatePolicy) echo.MiddlewareFunc
		pm      []echo.MiddlewareFunc
		opt     []echo.MiddlewareFunc
		admin   *echo.Group
		authCtl controller.IAuthController
	)
	e = echo.New()
	e.HideBanner = true

	for _, mw = range cfg.PreAuthMiddleware {
		e.Use(mw)
	}
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())

	rl = func(policy middleware.RatePolicy) echo.MiddlewareFunc {
		if cfg.RateLimit == nil {
			return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
		}
		return cfg.RateLimit.Middleware(policy)
	}

	if cfg.Health != nil {
		e.GET("/health", cfg.Health.Health)
	}

	if cfg.AuthUsecase != nil {
		var (
			secure    bool
			cookieCfg controller.CookieCfg
		)
		secure = os.Getenv("COOKIE_SECURE") == "true"
		cookieCfg = controller.NewCookieCfg(os.Getenv("REFRESH_COOKIE_NAME"), secure)
		authCtl = controller.NewAuthController(cfg.AuthUsecase, cookieCfg, cfg.AppPublicURL)
	}

	if authCtl != nil {
		e.POST("/auth/register", authCtl.Register, rl(middleware.PolicyRegisterBuyer))
		e.POST("/auth/register/seller", authCtl.RegisterSeller, rl(middleware.PolicyRegisterSeller))
		e.POST("/auth/login", authCtl.Login, rl(middleware.PolicyLogin))
		e.POST("/auth/forgot-password", authCtl.ForgotPassword, rl(middleware.PolicyForgotPassword))
		e.POST("/auth/reset-password", authCtl.ResetPassword, rl(middleware.PolicyResetPassword))
		e.GET("/auth/check-email", authCtl.CheckEmail, rl(middleware.PolicyCheckEmail))
		e.POST("/auth/refresh", authCtl.Refresh, rl(middleware.PolicyRefresh))
		e.POST("/auth/2fa/challenge", authCtl.Challenge2FA, rl(cfg.Policy2FAChallenge))

		if cfg.OAuthEnabled {
			e.GET("/auth/google", authCtl.GoogleStart, rl(middleware.PolicyGoogleOAuth))
			e.GET("/auth/google/callback", authCtl.GoogleCallback, rl(middleware.PolicyGoogleOAuth))
			e.POST("/auth/oauth/exchange", authCtl.OAuthExchange, rl(middleware.PolicyOAuthExchange))
			e.POST("/auth/oauth/link-confirm", authCtl.OAuthLinkConfirm, rl(middleware.PolicyOAuthExchange))
		}

		e.POST("/auth/logout", authCtl.Logout)
		pm = cfg.Protected
		e.POST("/auth/logout-all", authCtl.LogoutAllDevices, pm...)
		e.GET("/me", authCtl.Me, pm...)
		e.GET("/me/profile", authCtl.GetProfile, pm...)
		e.PUT("/me/profile", authCtl.UpdateProfile, pm...)
		e.POST("/me/change-password", authCtl.ChangePassword, pm...)
		e.POST("/auth/2fa/setup", authCtl.Setup2FA, pm...)
		e.POST("/auth/2fa/verify", authCtl.Verify2FA, pm...)
	}

	if cfg.Projects != nil {
		e.GET("/projects", cfg.Projects.ListPublic)
		e.GET("/projects/featured", cfg.Projects.ListFeatured)
		e.GET("/projects/mine", cfg.Projects.ListMine, cfg.SellerRead...)
		e.POST("/projects", cfg.Projects.Create, cfg.SellerWrite...)
		opt = cfg.OptionalJWT
		e.GET("/projects/:projectId/plans", cfg.Projects.ListPlans, opt...)
		e.GET("/projects/:projectId", cfg.Projects.Get, opt...)
		e.PUT("/projects/:projectId", cfg.Projects.Update, cfg.SellerWrite...)
		e.POST("/projects/:projectId/delete-request", cfg.Projects.RequestDelete, cfg.SellerWrite...)
	}

	if cfg.Connect != nil {
		e.GET("/seller/connect", cfg.Connect.GetAccount, cfg.SellerRead...)
		e.POST("/seller/connect/onboarding", cfg.Connect.CreateOnboarding, cfg.SellerWrite...)
	}

	if cfg.Seller != nil {
		e.GET("/seller/dashboard/stats", cfg.Seller.DashboardStats, cfg.SellerRead...)
		e.GET("/seller/supporters", cfg.Seller.ListSupporters, cfg.SellerRead...)
		e.GET("/seller/supporters/export", cfg.Seller.ExportSupporters, cfg.SellerRead...)
		e.GET("/seller/projects/analysis", cfg.Seller.ProjectAnalysis, cfg.SellerRead...)
	}

	if cfg.Payments != nil {
		pm = cfg.Protected
		e.POST("/payments/checkout", cfg.Payments.Checkout, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/payments/intent", cfg.Payments.Intent, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/payments/verify/checkout", cfg.Payments.VerifyCheckout, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/payments/verify/intent", cfg.Payments.VerifyIntent, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/stripe/webhook", cfg.Payments.Webhook)
		e.GET("/purchases", cfg.Payments.ListPurchases, cfg.Protected...)
	}

	if cfg.Supporters != nil {
		pm = cfg.Protected
		e.GET("/supporters", cfg.Supporters.List, pm...)
		e.POST("/supporters/:supporterId/upgrade", cfg.Supporters.Upgrade, pm...)
		e.POST("/supporters/:supporterId/upgrade-confirm", cfg.Supporters.UpgradeConfirm, pm...)
		e.POST("/supporters/:supporterId/cancel", cfg.Supporters.Cancel, pm...)
		e.POST("/supporters/:supporterId/rejoin", cfg.Supporters.Rejoin, pm...)
		e.POST("/supporters/:supporterId/billing/change", cfg.Supporters.ChangeBilling, pm...)
	}

	if cfg.Favorites != nil {
		pm = cfg.Protected
		e.GET("/favorites", cfg.Favorites.List, pm...)
		e.POST("/favorites", cfg.Favorites.Toggle, pm...)
		e.GET("/favorites/check/:projectId", cfg.Favorites.Check, pm...)
	}

	if cfg.Reports != nil {
		pm = cfg.Protected
		e.POST("/reports", cfg.Reports.Create, pm...)
	}

	if cfg.Media != nil {
		pm = cfg.Protected
		e.POST("/media/images", cfg.Media.UploadImage, append(pm, rl(middleware.PolicyMediaUpload))...)
	}

	if cfg.Messages != nil {
		pm = cfg.Protected
		e.GET("/messages", cfg.Messages.List, pm...)
		e.POST("/messages", cfg.Messages.Send, append(pm, rl(middleware.PolicyMessageSend))...)
		e.POST("/messages/:messageId/read", cfg.Messages.MarkRead, pm...)
		e.GET("/me/notifications", cfg.Messages.Notifications, pm...)
	}

	if cfg.WSToken != nil {
		pm = cfg.Protected
		e.POST("/ws/token", cfg.WSToken.Issue, append(pm, rl(middleware.PolicyWSToken))...)
	}

	if cfg.WS != nil && cfg.WSTokenAuth != nil {
		var wsMW []echo.MiddlewareFunc
		wsMW = append(wsMW, cfg.PreAuthMiddleware...)
		wsMW = append(wsMW, cfg.WSTokenAuth, rl(middleware.PolicyWSConnect))
		e.GET("/ws", cfg.WS.Handle, wsMW...)
	}

	if cfg.KYCFiles != nil {
		e.GET("/admin/kyc-documents", cfg.KYCFiles.Serve)
	}

	if cfg.Admin != nil || cfg.AdminOperations != nil {
		var (
			reviewAdmin      echo.MiddlewareFunc
			reportModeration echo.MiddlewareFunc
			userManagement   echo.MiddlewareFunc
			financeAdmin     echo.MiddlewareFunc
		)
		admin = e.Group("/admin", cfg.AdminOnly...)
		reportModeration = middleware.RequireAdminRole(
			model.AdminRoleSuperAdmin,
			model.AdminRoleReviewAdmin,
			model.AdminRoleSupportAdmin,
		)
		if cfg.Admin != nil {
			reviewAdmin = middleware.RequireAdminRole(model.AdminRoleReviewAdmin, model.AdminRoleSuperAdmin)
			userManagement = middleware.RequireAdminRole(model.AdminRoleSuperAdmin, model.AdminRoleSupportAdmin)
			financeAdmin = middleware.RequireAdminRole(model.AdminRoleSuperAdmin, model.AdminRoleFinanceAdmin)
			admin.GET("/stats", cfg.Admin.DashboardStats, userManagement)
			admin.GET("/users", cfg.Admin.ListUsers, userManagement)
			admin.PUT("/users/:userId", cfg.Admin.UpdateUser, userManagement)
			admin.DELETE("/users/:userId", cfg.Admin.DeleteUser, userManagement)
			admin.GET("/purchases", cfg.Admin.ListPurchases, financeAdmin)
			admin.PATCH("/purchases/:purchaseId", cfg.Admin.PatchPurchase, financeAdmin)
			admin.GET("/reviews/users", cfg.Admin.ListBuyerPending, reviewAdmin)
			admin.POST("/reviews/users/:userId/approve", cfg.Admin.ApproveBuyer, reviewAdmin)
			admin.POST("/reviews/users/:userId/reject", cfg.Admin.RejectBuyer, reviewAdmin)
			admin.GET("/seller-registrations", cfg.Admin.ListSellerRegs, reviewAdmin)
			admin.GET("/seller-registrations/:registrationId", cfg.Admin.GetSellerReg, reviewAdmin)
			admin.POST("/seller-registrations/:registrationId/approve", cfg.Admin.ApproveSellerReg, reviewAdmin)
			admin.POST("/seller-registrations/:registrationId/reject", cfg.Admin.RejectSellerReg, reviewAdmin)
			admin.GET("/projects", cfg.Admin.ListProjects, reviewAdmin)
			admin.PATCH("/projects/:projectId", cfg.Admin.PatchProject, reviewAdmin)
			admin.DELETE("/projects/:projectId", cfg.Admin.DeleteProject, reviewAdmin)
			admin.PATCH("/projects/:projectId/delete-request", cfg.Admin.PatchDeleteReq, reviewAdmin)
			admin.POST("/users/:userId/reset-2fa", cfg.Admin.ResetUser2FA,
				middleware.RequireAdminRole(model.AdminRoleSupportAdmin, model.AdminRoleSuperAdmin))
			admin.PATCH("/users/:userId/admin-role", cfg.Admin.ChangeAdminRole,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin))
		}
		if cfg.AdminOperations != nil {
			ops := cfg.AdminOperations
			admin.GET("/activity", ops.ListActivity,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin, model.AdminRoleSupportAdmin, model.AdminRoleFinanceAdmin))
			admin.GET("/security", ops.GetSecurity,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin, model.AdminRoleSupportAdmin))
			admin.GET("/batches", ops.ListBatches,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin, model.AdminRoleFinanceAdmin))
			admin.POST("/batches/:jobName/run", ops.RunBatch,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin))
			admin.GET("/audit-logs", ops.ListAuditLogs,
				middleware.RequireAdminRole(model.AdminRoleSuperAdmin))
		}
		if cfg.Reports != nil {
			admin.GET("/reports", cfg.Reports.ListAdmin, reportModeration)
			admin.PATCH("/reports/:reportId", cfg.Reports.PatchAdmin, reportModeration)
		}
	}

	return e
}
