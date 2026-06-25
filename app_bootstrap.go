package main

import (
	"database/sql"
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
	"crowdfund-app/repository"
	"crowdfund-app/router"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func buildApp() (*echo.Echo, string, error) {
	var (
		databaseURL    string
		err            error
		gormDB         *gorm.DB
		sqlDB          *sql.DB
		rdb            *redis.Client
		jwtSecret      string
		port           string
		publicRoot     string
		privateRoot    string
		userRepo       repository.UserRepo
		sellerRegRepo  repository.SellerRegRepo
		reTokenRepo    repository.ReTokenRepo
		resetRepo      repository.ResetRepo
		activityRepo   repository.ActivityRepo
		auditRepo           repository.AuditRepo
		reviewDecisionRepo  repository.ReviewDecisionRepo
		projectRepo    repository.ProjectRepo
		connectRepo    repository.ConnectRepo
		webhookRepo    repository.WebhookEventRepo
		supporterRepo  repository.SupporterRepo
		purchaseRepo   repository.PurchaseRepo
		oauthRepo      repository.OAuthRepo
		totpBackupRepo repository.TOTPBackupRepo
		oauthCache     repository.OAuthCache
		rateLimitRepo  repository.RateLimitRepo
		rateLimitUC    *usecase.RateLimitUseCase
		kycStore       *repository.LocalKYCStore
		tokenSvc       *auth.TokenService
		authUC         *usecase.AuthUseCase
		googleOAuth    *auth.GoogleOAuth
		stripeClient   stripe.Client
		cookieCfg      auth.CookieCfg
		authCtl        *controller.AuthController
		healthCtl      *controller.HealthController
		apiPublicURL   string
		adminUC        *usecase.AdminUseCase
		adminCtl       *controller.AdminController
		kycFileCtl     *controller.KYCFileController
		projectUC      *usecase.ProjectUseCase
		connectUC      *usecase.ConnectUseCase
		paymentUC      *usecase.PaymentUseCase
		projectCtl     *controller.ProjectController
		paymentCtl     *controller.PaymentController
		preAuth        []echo.MiddlewareFunc
		corsRaw        string
		origins        []string
		p              string
		s              string
		protected      []echo.MiddlewareFunc
		optionalJWT    []echo.MiddlewareFunc
		sellerRead     []echo.MiddlewareFunc
		sellerWrite    []echo.MiddlewareFunc
		adminOnly      []echo.MiddlewareFunc
		rateLimiter    *middleware.RateLimiter
		e              *echo.Echo
		validateSession func(echo.Context, auth.AuthClaims) error
	)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	databaseURL = os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
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
	_ = rdb

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
	reTokenRepo = repository.NewReTokenRepo(gormDB)
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
		Users:        userRepo,
		SellerRegs:   sellerRegRepo,
		ReTokens:     reTokenRepo,
		ResetToks:    resetRepo,
		Activity:     activityRepo,
		OAuth:        oauthRepo,
		OAuthCache:   oauthCache,
		TOTPBackup:   totpBackupRepo,
		Tokens:       tokenSvc,
		KYCStore:     kycStore,
		TOTPKey:      []byte(jwtSecret),
		TOTPIssuer:   strings.TrimSpace(os.Getenv("TOTP_ISSUER")),
		DB:           gormDB,
		AppPublicURL: os.Getenv("PUBLIC_APP_URL"),
	}
	googleOAuth, err = auth.GoogleFromEnv(auth.NewStateBridge(oauthCache))
	if err == nil {
		authUC.Google = googleOAuth
	} else {
		slog.Info("google oauth disabled", "reason", err.Error())
	}

	var stripeCfg stripe.Config
	stripeCfg, err = stripe.ConfigFromEnv()
	if err == nil {
		stripeClient = stripe.New(stripeCfg)
	} else {
		slog.Info("stripe disabled", "reason", err.Error())
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
		ReTokens:   reTokenRepo,
		TOTPBackup: totpBackupRepo,
		KYC:        kycStore,
		DB:         gormDB,
		APIBase:    apiPublicURL,
		SignSecret: []byte(jwtSecret),
	}
	adminCtl = controller.NewAdminController(adminUC)
	kycFileCtl = controller.NewKYCFileController(kycStore, []byte(jwtSecret))
	projectUC = &usecase.ProjectUseCase{Projects: projectRepo, Users: userRepo}
	connectUC = &usecase.ConnectUseCase{Users: userRepo, Connect: connectRepo, Stripe: stripeClient}
	paymentUC = &usecase.PaymentUseCase{
		DB:         gormDB,
		Users:      userRepo,
		Projects:   projectRepo,
		Purchases:  purchaseRepo,
		Supporters: supporterRepo,
		Connect:    connectRepo,
		Webhooks:   webhookRepo,
		Stripe:     stripeClient,
		ConnectUC:  connectUC,
	}
	projectCtl = controller.NewProjectController(projectUC)
	paymentCtl = controller.NewPaymentController(paymentUC)

	validateSession = func(c echo.Context, claims auth.AuthClaims) error {
		var (
			u   *entity.User
			err error
		)
		u, err = userRepo.FindByID(c.Request().Context(), claims.UserID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal error", "code": "INTERNAL_ERROR"})
		}
		if u == nil || !u.IsActive() {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "user inactive", "code": "USER_INACTIVE"})
		}
		if claims.TokenVersion != u.TokenVersion {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "token revoked", "code": "TOKEN_REVOKED"})
		}
		c.Set(middleware.CtxUserRole, string(u.Role))
		middleware.SetAdminRoleFromUser(c, u)
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
		Payments:           paymentCtl,
		Admin:              adminCtl,
		KYCFiles:           kycFileCtl,
		RateLimit:          rateLimiter,
		Policy2FAChallenge: middleware.Policy2FAChallenge([]byte(jwtSecret)),
		OAuthEnabled:      authUC.Google != nil,
		Protected:         protected,
		OptionalJWT:       optionalJWT,
		SellerRead:        sellerRead,
		SellerWrite:       sellerWrite,
		AdminOnly:         adminOnly,
	})
	return e, ":" + port, nil
}
