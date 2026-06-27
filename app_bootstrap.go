package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"crowdfund-app/auth"
	"crowdfund-app/controller"
	"crowdfund-app/db"
	"crowdfund-app/entity"
	"crowdfund-app/middleware"
	"crowdfund-app/bootstrap"
	"crowdfund-app/repository"
	"crowdfund-app/router"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func buildApp(opts *bootstrap.Options) (*echo.Echo, string, error) {
	var (
		databaseURL        string
		err                error
		gormDB             *gorm.DB
		sqlDB              *sql.DB
		rdb                *redis.Client
		jwtSecret          string
		port               string
		publicRoot         string
		privateRoot        string
		userRepo           repository.UserRepo
		sellerRegRepo      repository.SellerRegRepo
		refreshTokenRepo   repository.RefreshTokenRepo
		resetRepo          repository.ResetRepo
		activityRepo       repository.ActivityRepo
		auditRepo          repository.AuditRepo
		reviewDecisionRepo repository.ReviewDecisionRepo
		projectRepo        repository.ProjectRepo
		connectRepo        repository.ConnectRepo
		webhookRepo        repository.WebhookEventRepo
		supporterRepo      repository.SupporterRepo
		purchaseRepo       repository.PurchaseRepo
		oauthRepo          repository.OAuthRepo
		totpBackupRepo     repository.TOTPBackupRepo
		oauthCache         repository.OAuthCache
		rateLimitRepo      repository.RateLimitRepo
		rateLimitUC        *usecase.RateLimitUseCase
		kycStore           *repository.LocalKYCStore
		tokenSvc           *auth.TokenService
		authUC             *usecase.AuthUseCase
		googleOAuth        *auth.GoogleOAuth
		stripeClient       stripe.Client
		cookieCfg          auth.CookieCfg
		authCtl            *controller.AuthController
		healthCtl          *controller.HealthController
		apiPublicURL       string
		adminUC            *usecase.AdminUseCase
		adminCtl           *controller.AdminController
		kycFileCtl         *controller.KYCFileController
		projectUC          *usecase.ProjectUseCase
		connectUC          *usecase.ConnectUseCase
		paymentUC          *usecase.PaymentUseCase
		supporterUC        *usecase.SupporterUseCase
		purchaseUC         *usecase.PurchaseUseCase
		projectCtl         *controller.ProjectController
		connectCtl         *controller.ConnectController
		paymentCtl         *controller.PaymentController
		supporterCtl       *controller.SupporterController
		purchaseCtl        *controller.PurchaseController
		preAuth            []echo.MiddlewareFunc
		corsRaw            string
		origins            []string
		p                  string
		s                  string
		protected          []echo.MiddlewareFunc
		optionalJWT        []echo.MiddlewareFunc
		sellerRead         []echo.MiddlewareFunc
		sellerWrite        []echo.MiddlewareFunc
		adminOnly          []echo.MiddlewareFunc
		rateLimiter        *middleware.RateLimiter
		e                  *echo.Echo
		kycAccessUC        *usecase.KYCAccessUseCase
		validateSession    func(echo.Context, auth.AuthClaims) error
	)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	databaseURL = bootstrap.DatabaseURL()
	err = db.Up(databaseURL)
	if err != nil {
		return nil, "", err
	}
	gormDB, err = db.Open(databaseURL)
	if err != nil {
		return nil, "", err
	}
	sqlDB, err = gormDB.DB()
	if err != nil {
		return nil, "", err
	}
	err = sqlDB.Ping()
	if err != nil {
		return nil, "", err
	}

	rdb = db.OpenRedis()

	jwtSecret = os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return nil, "", fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	publicRoot = os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		publicRoot = "data/public"
	}
	err = os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755)
	if err != nil {
		return nil, "", err
	}

	privateRoot = os.Getenv("PRIVATE_ROOT")
	if privateRoot == "" {
		privateRoot = "data/private"
	}
	err = os.MkdirAll(filepath.Join(privateRoot, "kyc"), 0o700)
	if err != nil {
		return nil, "", err
	}

	userRepo = repository.NewUserRepo(gormDB)
	sellerRegRepo = repository.NewSellerRegRepo(gormDB)
	refreshTokenRepo = repository.NewRefreshTokenRepo(gormDB)
	resetRepo = repository.NewResetRepo(gormDB)
	activityRepo = repository.NewActivityRepo(gormDB)
	auditRepo = repository.NewAuditRepo(gormDB)
	reviewDecisionRepo = repository.NewReviewDecisionRepo(gormDB)
	projectRepo = repository.NewProjectRepo(gormDB)
	connectRepo = repository.NewConnectRepo(gormDB)
	webhookRepo = repository.NewWebhookEventRepo(gormDB)
	supporterRepo = repository.NewSupporterRepo(gormDB)
	purchaseRepo = repository.NewPurchaseRepo(gormDB)
	oauthRepo = repository.NewOAuthRepo(gormDB)
	totpBackupRepo = repository.NewTOTPBackupRepo(gormDB)
	oauthCache = repository.NewOAuthCache(rdb)
	rateLimitRepo = repository.NewRateLimitRepo(rdb)
	rateLimitUC = &usecase.RateLimitUseCase{Repo: rateLimitRepo}
	kycStore = repository.NewLocalKYC(privateRoot)

	tokenSvc = auth.NewTokenService([]byte(jwtSecret), auth.DefaultAuthTTL)
	authUC = &usecase.AuthUseCase{
		Users:         userRepo,
		SellerRegs:    sellerRegRepo,
		RefreshTokens: refreshTokenRepo,
		ResetToks:     resetRepo,
		Activity:      activityRepo,
		OAuth:         oauthRepo,
		OAuthCache:    oauthCache,
		TOTPBackup:    totpBackupRepo,
		Tokens:        tokenSvc,
		KYCStore:      kycStore,
		TOTPKey:       []byte(jwtSecret),
		TOTPIssuer:    strings.TrimSpace(os.Getenv("TOTP_ISSUER")),
		DB:            gormDB,
		AppPublicURL:  os.Getenv("PUBLIC_APP_URL"),
	}
	googleOAuth, err = auth.GoogleFromEnv(auth.NewStateBridge(oauthCache))
	if err == nil {
		authUC.Google = googleOAuth
	} else {
		slog.Info("google oauth disabled", "reason", err.Error())
	}

	if opts != nil && opts.Stripe != nil {
		stripeClient = opts.Stripe
	} else {
		stripeClient = bootstrap.StripeFromEnv()
	}

	cookieCfg = auth.CookieFromEnv()
	authCtl = controller.NewAuthController(authUC, cookieCfg)
	healthCtl = &controller.HealthController{SQL: gormDB, RDB: rdb}

	apiPublicURL = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if apiPublicURL == "" {
		apiPublicURL = "http://localhost:" + port
	}
	adminUC = &usecase.AdminUseCase{
		Users:           userRepo,
		SellerRegs:      sellerRegRepo,
		Projects:        projectRepo,
		Audit:           auditRepo,
		ReviewDecisions: reviewDecisionRepo,
		RefreshTokens:   refreshTokenRepo,
		TOTPBackup:      totpBackupRepo,
		KYC:             kycStore,
		DB:              gormDB,
		APIBase:         apiPublicURL,
		SignSecret:      []byte(jwtSecret),
	}
	adminCtl = controller.NewAdminController(adminUC)
	kycAccessUC = &usecase.KYCAccessUseCase{KYC: kycStore, SignSecret: []byte(jwtSecret)}
	kycFileCtl = controller.NewKYCFileController(kycAccessUC)
	projectUC = &usecase.ProjectUseCase{Projects: projectRepo, Users: userRepo, DB: gormDB}
	connectUC = &usecase.ConnectUseCase{Users: userRepo, Connect: connectRepo, Stripe: stripeClient}
	supporterUC = bootstrap.NewSupporterUseCase(gormDB, bootstrap.Repositories{
		Users:      userRepo,
		Projects:   projectRepo,
		Supporters: supporterRepo,
		Purchases:  purchaseRepo,
		Connect:    connectRepo,
		Activity:   activityRepo,
	}, stripeClient)
	paymentUC = &usecase.PaymentUseCase{
		DB:              gormDB,
		Users:           userRepo,
		Projects:        projectRepo,
		Purchases:       purchaseRepo,
		Supporters:      supporterRepo,
		Connect:         connectRepo,
		Webhooks:        webhookRepo,
		Activity:        activityRepo,
		Stripe:          stripeClient,
		ConnectSync:     connectUC,
		SupporterViewer: supporterUC,
	}
	purchaseUC = &usecase.PurchaseUseCase{
		Users:     userRepo,
		Projects:  projectRepo,
		Purchases: purchaseRepo,
	}
	projectCtl = controller.NewProjectController(projectUC)
	connectCtl = controller.NewConnectController(connectUC)
	paymentCtl = controller.NewPaymentController(paymentUC)
	supporterCtl = controller.NewSupporterController(supporterUC)
	purchaseCtl = controller.NewPurchaseController(purchaseUC)

	validateSession = func(c echo.Context, claims auth.AuthClaims) error {
		var (
			info usecase.SessionInfo
			err  error
		)
		info, err = authUC.ValidateSession(c.Request().Context(), claims)
		if err != nil {
			switch {
			case errors.Is(err, usecase.ErrInactiveUser):
				return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "user inactive", "code": "USER_INACTIVE"})
			case errors.Is(err, usecase.ErrTokenRevoked):
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "token revoked", "code": "TOKEN_REVOKED"})
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
			}
		}
		c.Set(middleware.CtxUserRole, string(info.Role))
		middleware.SetAdminRoleFromSession(c, info.Role, info.AdminRole)
		return nil
	}

	corsRaw = strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		for _, p = range strings.Split(corsRaw, ",") {
			s = strings.TrimSpace(p)
			if s != "" {
				origins = append(origins, s)
			}
		}
		if len(origins) > 0 {
			preAuth = append(preAuth, labmw.CORSWithConfig(labmw.CORSConfig{
				AllowOrigins:     origins,
				AllowCredentials: true,
				AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
				AllowMethods: []string{
					http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
					http.MethodPost, http.MethodDelete, http.MethodOptions,
				},
			}))
		}
	}

	protected = []echo.MiddlewareFunc{
		middleware.JWTAuth(authUC.ParseToken, validateSession),
	}
	optionalJWT = []echo.MiddlewareFunc{
		middleware.OptionalJWTAuth(authUC.ParseToken, validateSession),
	}
	sellerRead = append(append([]echo.MiddlewareFunc{}, protected...), middleware.RequireRole(entity.UserRoleSeller))
	sellerWrite = append(append([]echo.MiddlewareFunc{}, sellerRead...), middleware.Require2FA())
	adminOnly = append(
		append([]echo.MiddlewareFunc{}, protected...),
		middleware.RequireRole(entity.UserRoleAdmin),
		middleware.Require2FA(),
	)
	rateLimiter = middleware.NewRateLimiter(rateLimitUC)

	e = router.New(router.Config{
		PreAuthMiddleware:  preAuth,
		Health:             healthCtl,
		Auth:               authCtl,
		Projects:           projectCtl,
		Connect:            connectCtl,
		Payments:           paymentCtl,
		Supporters:         supporterCtl,
		Purchases:          purchaseCtl,
		Admin:              adminCtl,
		KYCFiles:           kycFileCtl,
		RateLimit:          rateLimiter,
		Policy2FAChallenge: middleware.Policy2FAChallenge([]byte(jwtSecret)),
		OAuthEnabled:       authUC.Google != nil,
		Protected:          protected,
		OptionalJWT:        optionalJWT,
		SellerRead:         sellerRead,
		SellerWrite:        sellerWrite,
		AdminOnly:          adminOnly,
	})
	return e, ":" + port, nil
}
