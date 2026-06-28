package router

import (
	"crowdfund-app/controller"
	"crowdfund-app/entity"
	"crowdfund-app/middleware"

	"github.com/labstack/echo/v4"
)

type Config struct {
	Health            *controller.HealthController
	Auth              *controller.AuthController
	Projects          *controller.ProjectController
	Connect           *controller.ConnectController
	Payments          *controller.PaymentController
	Supporters        *controller.SupporterController
	Messages          *controller.MessageController
	WSToken           *controller.WSTokenController
	WS                *controller.WSController
	Favorites         *controller.FavoriteController
	Reports           *controller.ReportController
	Media             *controller.MediaController
	Admin             *controller.AdminController
	AdminOperations   *controller.AdminOperationsController
	KYCFiles          *controller.KYCFileController
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
		e     *echo.Echo
		mw    echo.MiddlewareFunc
		rl    func(policy middleware.RatePolicy) echo.MiddlewareFunc
		pm    []echo.MiddlewareFunc
		opt   []echo.MiddlewareFunc
		admin *echo.Group
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

	if cfg.Auth != nil {
		e.POST("/auth/register", cfg.Auth.Register, rl(middleware.PolicyRegisterBuyer))
		e.POST("/auth/register/seller", cfg.Auth.RegisterSeller, rl(middleware.PolicyRegisterSeller))
		e.POST("/auth/login", cfg.Auth.Login, rl(middleware.PolicyLogin))
		e.POST("/auth/forgot-password", cfg.Auth.ForgotPassword, rl(middleware.PolicyForgotPassword))
		e.POST("/auth/reset-password", cfg.Auth.ResetPassword, rl(middleware.PolicyResetPassword))
		e.GET("/auth/check-email", cfg.Auth.CheckEmail, rl(middleware.PolicyCheckEmail))
		e.POST("/auth/refresh", cfg.Auth.Refresh, rl(middleware.PolicyRefresh))
		e.POST("/auth/2fa/challenge", cfg.Auth.Challenge2FA, rl(cfg.Policy2FAChallenge))

		if cfg.OAuthEnabled {
			e.GET("/auth/google", cfg.Auth.GoogleStart, rl(middleware.PolicyGoogleOAuth))
			e.GET("/auth/google/callback", cfg.Auth.GoogleCallback, rl(middleware.PolicyGoogleOAuth))
			e.POST("/auth/oauth/exchange", cfg.Auth.OAuthExchange, rl(middleware.PolicyOAuthExchange))
		}

		e.POST("/auth/logout", cfg.Auth.Logout)
		pm = cfg.Protected
		e.GET("/me", cfg.Auth.Me, pm...)
		e.POST("/me/change-password", cfg.Auth.ChangePassword, pm...)
		e.POST("/auth/2fa/setup", cfg.Auth.Setup2FA, pm...)
		e.POST("/auth/2fa/verify", cfg.Auth.Verify2FA, pm...)
	}

	if cfg.Projects != nil {
		e.GET("/projects", cfg.Projects.ListPublic)
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
		)
		admin = e.Group("/admin", cfg.AdminOnly...)
		reportModeration = middleware.RequireAdminRole(
			entity.AdminRoleSuperAdmin,
			entity.AdminRoleReviewAdmin,
			entity.AdminRoleSupportAdmin,
		)
		if cfg.Admin != nil {
			reviewAdmin = middleware.RequireAdminRole(entity.AdminRoleReviewAdmin, entity.AdminRoleSuperAdmin)
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
				middleware.RequireAdminRole(entity.AdminRoleSupportAdmin, entity.AdminRoleSuperAdmin))
			admin.PATCH("/users/:userId/admin-role", cfg.Admin.ChangeAdminRole,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin))
		}
		if cfg.AdminOperations != nil {
			ops := cfg.AdminOperations
			admin.GET("/activity", ops.ListActivity,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin, entity.AdminRoleSupportAdmin, entity.AdminRoleFinanceAdmin))
			admin.GET("/security", ops.GetSecurity,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin, entity.AdminRoleSupportAdmin))
			admin.GET("/batches", ops.ListBatches,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin, entity.AdminRoleFinanceAdmin))
			admin.POST("/batches/:jobName/run", ops.RunBatch,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin))
			admin.GET("/audit-logs", ops.ListAuditLogs,
				middleware.RequireAdminRole(entity.AdminRoleSuperAdmin))
		}
		if cfg.Reports != nil {
			admin.GET("/reports", cfg.Reports.ListAdmin, reportModeration)
			admin.PATCH("/reports/:reportId", cfg.Reports.PatchAdmin, reportModeration)
		}
	}

	return e
}
