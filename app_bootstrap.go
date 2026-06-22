package main

import (
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
	"crowdfund-app/storage"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
)

func buildApp() (*echo.Echo, string, error) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	if err := db.Up(databaseURL); err != nil {
		return nil, "", err
	}
	gormDB, err := db.Open(databaseURL)
	if err != nil {
		return nil, "", err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, "", err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, "", err
	}

	rdb := db.OpenRedis()
	_ = rdb

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return nil, "", fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	publicRoot := os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		publicRoot = "data/public"
	}
	if err := os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755); err != nil {
		return nil, "", err
	}

	privateRoot := os.Getenv("PRIVATE_ROOT")
	if privateRoot == "" {
		privateRoot = "data/private"
	}
	if err := os.MkdirAll(filepath.Join(privateRoot, "kyc"), 0o700); err != nil {
		return nil, "", err
	}

	userRepo := repository.NewUserRepo(gormDB)
	sellerRegRepo := repository.NewSellerRegRepo(gormDB)
	reTokenRepo := repository.NewReTokenRepo(gormDB)
	resetRepo := repository.NewResetRepo(gormDB)
	activityRepo := repository.NewActivityRepo(gormDB)
	auditRepo := repository.NewAuditRepo(gormDB)
	projectRepo := repository.NewProjectRepo(gormDB)
	oauthRepo := repository.NewOAuthRepo(gormDB)
	totpBackupRepo := repository.NewTOTPBackupRepo(gormDB)
	oauthCache := repository.NewOAuthCache(rdb)
	rateLimitRepo := repository.NewRateLimitRepo(rdb)
	kycStore := storage.NewLocalKYC(privateRoot)

	tokenSvc := auth.NewTokenService([]byte(jwtSecret), auth.DefaultAuthTTL)
	authUC := &usecase.AuthUseCase{
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
	if googleOAuth, err := auth.GoogleFromEnv(auth.NewStateBridge(oauthCache)); err == nil {
		authUC.Google = googleOAuth
	} else {
		slog.Info("google oauth disabled", "reason", err.Error())
	}

	cookieCfg := auth.CookieFromEnv()
	authCtl := controller.NewAuthController(authUC, cookieCfg)
	healthCtl := &controller.HealthController{SQL: gormDB, RDB: rdb}

	apiPublicURL := strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if apiPublicURL == "" {
		apiPublicURL = "http://localhost:" + port
	}
	adminUC := &usecase.AdminUseCase{
		Users:      userRepo,
		SellerRegs: sellerRegRepo,
		Projects:   projectRepo,
		Audit:      auditRepo,
		KYC:        kycStore,
		DB:         gormDB,
		APIBase:    apiPublicURL,
		SignSecret: []byte(jwtSecret),
	}
	adminCtl := controller.NewAdminController(adminUC)
	kycFileCtl := controller.NewKYCFileController(kycStore, []byte(jwtSecret))
	projectUC := &usecase.ProjectUseCase{Projects: projectRepo, Users: userRepo}
	projectCtl := controller.NewProjectController(projectUC)

	validateSession := func(c echo.Context, claims auth.AuthClaims) error {
		u, err := userRepo.FindByID(c.Request().Context(), claims.UserID)
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
		return nil
	}

	var preAuth []echo.MiddlewareFunc
	corsRaw := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		var origins []string
		for _, p := range strings.Split(corsRaw, ",") {
			if s := strings.TrimSpace(p); s != "" {
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

	protected := []echo.MiddlewareFunc{
		middleware.JWTAuth(authUC.ParseToken, validateSession),
	}
	optionalJWT := []echo.MiddlewareFunc{
		middleware.OptionalJWTAuth(authUC.ParseToken, validateSession),
	}
	sellerRead := append(append([]echo.MiddlewareFunc{}, protected...), middleware.RequireRole(entity.UserRoleSeller))
	sellerWrite := append(append([]echo.MiddlewareFunc{}, sellerRead...), middleware.Require2FA())
	adminOnly := append(
		append([]echo.MiddlewareFunc{}, protected...),
		middleware.RequireRole(entity.UserRoleAdmin),
		middleware.Require2FA(),
	)
	rateLimiter := middleware.NewRateLimiter(rateLimitRepo)

	e := router.New(router.Config{
		PreAuthMiddleware: preAuth,
		Health:            healthCtl,
		Auth:              authCtl,
		Projects:          projectCtl,
		Admin:             adminCtl,
		KYCFiles:          kycFileCtl,
		RateLimit:         rateLimiter,
		OAuthEnabled:      authUC.Google != nil,
		Protected:         protected,
		OptionalJWT:       optionalJWT,
		SellerRead:        sellerRead,
		SellerWrite:       sellerWrite,
		AdminOnly:         adminOnly,
	})
	return e, ":" + port, nil
}
