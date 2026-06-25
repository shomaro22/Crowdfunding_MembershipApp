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
	Payments          *controller.PaymentController
	Admin             *controller.AdminController
	KYCFiles          *controller.KYCFileController
	RateLimit         *middleware.RateLimiter
	Policy2FAChallenge middleware.RatePolicy
	OAuthEnabled      bool
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
		e.POST("/auth/reset-password", cfg.Auth.ResetPassword)
		e.GET("/auth/check-email", cfg.Auth.CheckEmail)
		e.POST("/re", cfg.Auth.Re, rl(middleware.PolicyRe))
		e.POST("/auth/2fa/challenge", cfg.Auth.Challenge2FA, rl(cfg.Policy2FAChallenge))

		if cfg.OAuthEnabled {
			e.GET("/auth/google", cfg.Auth.GoogleStart, rl(middleware.PolicyGoogleOAuth))
			e.GET("/auth/google/callback", cfg.Auth.GoogleCallback, rl(middleware.PolicyGoogleOAuth))
			e.POST("/auth/oauth/exchange", cfg.Auth.OAuthExchange, rl(middleware.PolicyOAuthExchange))
		}

		pm = cfg.Protected
		e.POST("/auth/logout", cfg.Auth.Logout, pm...)
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
		e.GET("/projects/:projectId/plans", cfg.Projects.ListPlans)
		e.GET("/projects/:projectId", cfg.Projects.Get, opt...)
		e.PUT("/projects/:projectId", cfg.Projects.Update, cfg.SellerWrite...)
		e.POST("/projects/:projectId/delete-request", cfg.Projects.RequestDelete, cfg.SellerWrite...)
	}

	if cfg.Payments != nil {
		pm = cfg.Protected
		e.POST("/payments/checkout", cfg.Payments.Checkout, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/payments/intent", cfg.Payments.Intent, append(pm, rl(middleware.PolicyPayment))...)
		e.POST("/stripe/webhook", cfg.Payments.Webhook)
	}

	if cfg.KYCFiles != nil {
		e.GET("/admin/kyc-documents", cfg.KYCFiles.Serve)
	}

	if cfg.Admin != nil && len(cfg.AdminOnly) > 0 {
		var reviewAdmin echo.MiddlewareFunc
		admin = e.Group("/admin", cfg.AdminOnly...)
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

	return e
}
