//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crowdfund-app/controller"
	"crowdfund-app/db"
	"crowdfund-app/middleware"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/router"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"
	"crowdfund-app/wshub"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type appConfig struct {
	JWTSecret         string
	TOTPEncryptionKey []byte
	KYCSigningKey     []byte
	Pending2FAKey     []byte
	Port              string
	PublicRoot        string
	PrivateRoot       string
	APIPublicURL      string
	AppPublicURL      string
	TOTPIssuer        string
}

func secretKey(envKey, fallback string) []byte {
	v := strings.TrimSpace(os.Getenv(envKey))
	if len(v) >= 32 {
		return []byte(v)
	}
	return []byte(fallback)
}

func loadConfig() (appConfig, error) {
	var (
		cfg appConfig
		err error
	)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if len(cfg.JWTSecret) < 32 {
		return appConfig{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	cfg.TOTPEncryptionKey = secretKey("TOTP_ENCRYPTION_KEY", cfg.JWTSecret)
	cfg.KYCSigningKey = secretKey("KYC_SIGNING_KEY", cfg.JWTSecret)
	cfg.Pending2FAKey = secretKey("PENDING_2FA_KEY", cfg.JWTSecret)
	cfg.Port = os.Getenv("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	cfg.PublicRoot = os.Getenv("PUBLIC_ROOT")
	if cfg.PublicRoot == "" {
		cfg.PublicRoot = "data/public"
	}
	cfg.PrivateRoot = os.Getenv("PRIVATE_ROOT")
	if cfg.PrivateRoot == "" {
		cfg.PrivateRoot = "data/private"
	}
	err = os.MkdirAll(filepath.Join(cfg.PublicRoot, "uploads"), 0o755)
	if err != nil {
		return appConfig{}, err
	}
	err = os.MkdirAll(filepath.Join(cfg.PrivateRoot, "kyc"), 0o700)
	if err != nil {
		return appConfig{}, err
	}
	cfg.APIPublicURL = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
	if cfg.APIPublicURL == "" {
		cfg.APIPublicURL = "http://localhost:" + cfg.Port
	}
	cfg.AppPublicURL = os.Getenv("PUBLIC_APP_URL")
	cfg.TOTPIssuer = strings.TrimSpace(os.Getenv("TOTP_ISSUER"))
	return cfg, nil
}

func (c appConfig) listenAddr() string {
	return ":" + c.Port
}

func databaseURL() string {
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
}

func LoadDotEnv() {
	const envFile = ".env"
	if _, err := os.Stat(envFile); err != nil {
		return
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}

func stripeFromEnv() stripe.Client {
	cfg, err := stripe.ConfigFromEnv()
	if err != nil {
		slog.Info("stripe disabled", "reason", err.Error())
		return nil
	}
	return stripe.New(cfg)
}

func openDB() (*gorm.DB, error) {
	if err := db.Up(databaseURL()); err != nil {
		return nil, err
	}
	gormDB, err := db.Open(databaseURL())
	if err != nil {
		return nil, err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}
	return gormDB, nil
}

type Options struct {
	Stripe stripe.Client
	DB     *gorm.DB
	Redis  *redis.Client
}

func BuildHTTP(opts *Options) (*echo.Echo, string, error) {
	var (
		err             error
		cfg             appConfig
		healthCtl       controller.IHealthController
		adminCtl        controller.IAdminController
		adminOpsCtl     controller.IAdminOperationsController
		kycFileCtl      controller.IKYCFileController
		projectCtl      controller.IProjectController
		connectCtl      controller.IConnectController
		paymentCtl      controller.IPaymentController
		supporterCtl    controller.ISupporterController
		favoriteCtl     controller.IFavoriteController
		reportCtl       controller.IReportController
		mediaCtl        controller.IMediaController
		sellerCtl       controller.ISellerController
		messageCtl      controller.IMessageController
		wsTokenCtl      controller.IWSTokenController
		wsCtl           controller.IWSController
		preAuth         []echo.MiddlewareFunc
		corsRaw         string
		origins         []string
		p               string
		s               string
		protected       []echo.MiddlewareFunc
		optionalJWT     []echo.MiddlewareFunc
		sellerRead      []echo.MiddlewareFunc
		sellerWrite     []echo.MiddlewareFunc
		adminOnly       []echo.MiddlewareFunc
		rateLimiter     *middleware.RateLimiter
		e               *echo.Echo
		validateSession middleware.SessionValidator
		gormDB          *gorm.DB
		rdb             *redis.Client
		stripeClient    stripe.Client
		users           repository.IUserRepo
		sellerRegs      repository.ISellerRegRepo
		refreshTokens   repository.IRefreshTokenRepo
		resetToks       repository.IResetRepo
		activity        repository.IActivityRepo
		audit           repository.IAuditRepo
		reviewDecisions repository.IReviewDecisionRepo
		projects        repository.IProjectRepo
		connect         repository.IConnectRepo
		webhooks        repository.IWebhookEventRepo
		supporters      repository.ISupporterRepo
		purchases       repository.IPurchaseRepo
		oauth           repository.IOAuthRepo
		totpBackup      repository.ITOTPBackupRepo
		batch           repository.IBatchRepo
		messages        repository.IMessageRepo
		wsTokens        repository.IWSTokenRepo
		favorites       repository.IFavoriteRepo
		reports         repository.IReportRepo
		seller          repository.ISellerRepo
		adminStats      repository.IAdminRepo
		oauthCache      repository.IOAuthCache
		rateLimit       repository.IRateLimitRepo
		kycStore        *repository.LocalKYCStore
		mediaStore      *repository.LocalMediaStore
		hub             *wshub.ConnectionManager
		authUC          usecase.IAuthUsecase
		projectUCRaw    *usecase.ProjectUseCase
		projectUC       usecase.IProjectUsecase
		kycAccessRaw    *usecase.KYCAccessUseCase
		kycAccess       usecase.IKYCAccessUsecase
		connectUCRaw    *usecase.ConnectUseCase
		connectUC       usecase.IConnectUsecase
		supportUCRaw    *usecase.SupporterUseCase
		supportUC       usecase.ISupporterUsecase
		purchaseUCRaw   *usecase.PurchaseUseCase
		purchaseUC      usecase.IPurchaseUsecase
		paymentUC       usecase.IPaymentUsecase
		adminUC         usecase.IAdminUsecase
		msgUCRaw        *usecase.MessageUseCase
		msgUC           usecase.IMessageUsecase
		wsTokenUC       usecase.IWSTokenUsecase
		opsUC           usecase.IOperationsUsecase
		favUC           usecase.IFavoriteUsecase
		reportUC        usecase.IReportUsecase
		mediaUC         usecase.IMediaUsecase
		sellerUC        usecase.ISellerUsecase
		rateLimitUC     usecase.IRateLimitUsecase
		batchUC         *usecase.BatchUseCase
	)

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err = loadConfig()
	if err != nil {
		return nil, "", err
	}
	if opts != nil && opts.DB != nil {
		gormDB = opts.DB
	} else {
		gormDB, err = openDB()
		if err != nil {
			return nil, "", err
		}
	}
	if opts != nil && opts.Redis != nil {
		rdb = opts.Redis
	} else {
		rdb = db.OpenRedis()
	}
	if opts != nil && opts.Stripe != nil {
		stripeClient = opts.Stripe
	} else {
		stripeClient = stripeFromEnv()
	}

	users = repository.NewUserRepo(gormDB)
	sellerRegs = repository.NewSellerRegRepo(gormDB)
	refreshTokens = repository.NewRefreshTokenRepo(gormDB)
	resetToks = repository.NewResetRepo(gormDB)
	activity = repository.NewActivityRepo(gormDB)
	audit = repository.NewAuditRepo(gormDB)
	reviewDecisions = repository.NewReviewDecisionRepo(gormDB)
	projects = repository.NewProjectRepo(gormDB)
	connect = repository.NewConnectRepo(gormDB)
	webhooks = repository.NewWebhookEventRepo(gormDB)
	supporters = repository.NewSupporterRepo(gormDB)
	purchases = repository.NewPurchaseRepo(gormDB)
	oauth = repository.NewOAuthRepo(gormDB)
	totpBackup = repository.NewTOTPBackupRepo(gormDB)
	batch = repository.NewBatchRepo(gormDB)
	messages = repository.NewMessageRepo(gormDB)
	wsTokens = repository.NewWSTokenRepo(gormDB)
	favorites = repository.NewFavoriteRepo(gormDB)
	reports = repository.NewReportRepo(gormDB)
	seller = repository.NewSellerRepo(gormDB)
	adminStats = repository.NewAdminRepo(gormDB)
	oauthCache = repository.NewOAuthCache(rdb)
	rateLimit = repository.NewRateLimitRepo(rdb)
	kycStore = repository.NewLocalKYC(cfg.PrivateRoot)
	mediaStore = repository.NewLocalMedia(cfg.PublicRoot)
	hub = wshub.NewConnectionManager()

	authUC = usecase.NewAuthUsecase(usecase.AuthConfig{
		Users: users, SellerRegs: sellerRegs, RefreshTokens: refreshTokens, ResetToks: resetToks,
		Activity: activity, OAuth: oauth, OAuthCache: oauthCache, TOTPBackup: totpBackup,
		KYCStore: kycStore, JWTSecret: cfg.JWTSecret, TOTPEncryptionKey: cfg.TOTPEncryptionKey,
		Pending2FAKey: cfg.Pending2FAKey, TOTPIssuer: cfg.TOTPIssuer, AppPublicURL: cfg.AppPublicURL, DB: gormDB,
		Mailer: usecase.NewLogMailer(cfg.AppPublicURL),
	})
	projectUCRaw = usecase.NewProjectUsecase(projects, users, favorites, gormDB)
	projectUC = projectUCRaw
	kycAccessRaw = usecase.NewKYCAccessUsecase(kycStore, string(cfg.KYCSigningKey))
	kycAccess = kycAccessRaw
	connectUCRaw = usecase.NewConnectUsecase(users, connect, stripeClient)
	connectUC = connectUCRaw
	supportUCRaw = usecase.NewSupporterUsecase(gormDB, users, projects, supporters, purchases, connect, activity, stripeClient)
	supportUC = supportUCRaw
	purchaseUCRaw = usecase.NewPurchaseUsecase(users, projects, purchases)
	purchaseUC = purchaseUCRaw
	paymentUC = usecase.NewPaymentUsecase(gormDB, users, projects, purchases, supporters, connect, webhooks, activity, stripeClient, connectUCRaw, supportUCRaw)
	batchUC = usecase.NewBatchUsecase(gormDB, batch, supportUCRaw, wsTokens)
	adminUC = usecase.NewAdminUsecase(gormDB, usecase.AdminRepos{
		Users: users, SellerRegs: sellerRegs, Projects: projects, Audit: audit,
		ReviewDecisions: reviewDecisions, RefreshTokens: refreshTokens, TOTPBackup: totpBackup,
		Admin: adminStats, Purchases: purchases,
	}, kycStore, projectUCRaw, kycAccessRaw, purchaseUCRaw)
	msgUCRaw = usecase.NewMessageUsecase(messages, projects, supporters, users, activity)
	wireMessageEvents(msgUCRaw, hub)
	msgUC = msgUCRaw
	wsTokenUC = usecase.NewWSTokenUsecase(wsTokens, users, activity)
	opsUC = usecase.NewOperationsUsecase(activity, audit, batch, users, batchUC)
	favUC = usecase.NewFavoriteUsecase(favorites, projectUCRaw, activity)
	reportUC = usecase.NewReportUsecase(reports, users, projects, messages, activity, audit)
	mediaUC = usecase.NewMediaUsecase(mediaStore, cfg.APIPublicURL)
	sellerUC = usecase.NewSellerUsecase(projects, seller, supportUCRaw)
	rateLimitUC = usecase.NewRateLimitUsecase(rateLimit)

	corsRaw = strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw != "" {
		for _, p = range strings.Split(corsRaw, ",") {
			s = strings.TrimSpace(p)
			if s != "" {
				origins = append(origins, s)
			}
		}
	}

	healthCtl = controller.NewHealthController(gormDB, rdb)
	adminCtl = controller.NewAdminController(adminUC, cfg.APIPublicURL)
	adminOpsCtl = controller.NewAdminOperationsController(opsUC)
	kycFileCtl = controller.NewKYCFileController(kycAccess)
	projectCtl = controller.NewProjectController(projectUC)
	connectCtl = controller.NewConnectController(connectUC)
	if stripeClient != nil {
		paymentCtl = controller.NewPaymentController(paymentUC, purchaseUC, stripeClient.AppURL())
	}
	supporterCtl = controller.NewSupporterController(supportUC)
	favoriteCtl = controller.NewFavoriteController(favUC)
	reportCtl = controller.NewReportController(reportUC)
	mediaCtl = controller.NewMediaController(mediaUC)
	sellerCtl = controller.NewSellerController(sellerUC)
	messageCtl = controller.NewMessageController(msgUC)
	wsTokenCtl = controller.NewWSTokenController(wsTokenUC)
	wsCtl = controller.NewWSController(hub, msgUC, origins)

	validateSession = middleware.SessionValidateHTTP(authUC.ValidateSession)

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

	protected = []echo.MiddlewareFunc{
		middleware.JWTAuth(authUC.ParseToken, validateSession),
	}
	optionalJWT = []echo.MiddlewareFunc{
		middleware.OptionalJWTAuth(authUC.ParseToken, validateSession),
	}
	sellerRead = append(append([]echo.MiddlewareFunc{}, protected...), middleware.RequireRole(model.UserRoleSeller))
	sellerWrite = append(append([]echo.MiddlewareFunc{}, sellerRead...), middleware.Require2FA())
	adminOnly = append(
		append([]echo.MiddlewareFunc{}, protected...),
		middleware.RequireRole(model.UserRoleAdmin),
		middleware.Require2FA(),
	)
	rateLimiter = middleware.NewRateLimiter(rateLimitUC)
	wireRateLimitActivity(activity, rateLimiter)

	e = router.New(router.Config{
		PreAuthMiddleware:  preAuth,
		Health:             healthCtl,
		AuthUsecase:        authUC,
		AppPublicURL:       cfg.AppPublicURL,
		Projects:           projectCtl,
		Connect:            connectCtl,
		Seller:             sellerCtl,
		Payments:           paymentCtl,
		Supporters:         supporterCtl,
		Favorites:          favoriteCtl,
		Reports:            reportCtl,
		Media:              mediaCtl,
		Messages:           messageCtl,
		WSToken:            wsTokenCtl,
		WS:                 wsCtl,
		WSTokenAuth:        middleware.WSTokenAuth(wsTokenUC.Consume),
		Admin:              adminCtl,
		AdminOperations:    adminOpsCtl,
		KYCFiles:           kycFileCtl,
		RateLimit:          rateLimiter,
		Policy2FAChallenge: middleware.Policy2FAChallenge(cfg.Pending2FAKey),
		OAuthEnabled:       authUC.OAuthEnabled(),
		Protected:          protected,
		OptionalJWT:        optionalJWT,
		SellerRead:         sellerRead,
		SellerWrite:        sellerWrite,
		AdminOnly:          adminOnly,
	})
	e.Static("/uploads", filepath.Join(cfg.PublicRoot, "uploads"))
	return e, cfg.listenAddr(), nil
}

func wireRateLimitActivity(activity repository.IActivityRepo, limiter *middleware.RateLimiter) {
	if limiter == nil {
		return
	}
	limiter.OnDenied = func(c echo.Context, policy middleware.RatePolicy, ruleKey string) {
		var (
			uid    *uuid.UUID
			userID uuid.UUID
			ok     bool
			meta   usecase.RequestMeta
		)
		if userID, ok = middleware.AuthUserID(c); ok {
			uid = &userID
		}
		meta = httpRequestMeta(c)
		usecase.LogRateLimitDenied(c.Request().Context(), activity, uid, policy.Name, ruleKey, meta)
	}
}

func httpRequestMeta(c echo.Context) usecase.RequestMeta {
	reqID, _ := c.Get(middleware.CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}

func wireMessageEvents(msg *usecase.MessageUseCase, hub *wshub.ConnectionManager) {
	if msg == nil || hub == nil {
		return
	}
	usecase.AttachMessageEvents(msg, newMessageEventBridge(hub))
}

type messageEventBridge struct {
	pub *wshub.MessagePublisher
}

func newMessageEventBridge(hub *wshub.ConnectionManager) *messageEventBridge {
	return &messageEventBridge{pub: wshub.NewMessagePublisher(hub)}
}

func (b *messageEventBridge) OnMessageCreated(_ context.Context, view usecase.MessageItemView) {
	if b == nil || b.pub == nil {
		return
	}
	b.pub.PublishMessageCreated(view.ProjectID, messageItemPayloadFrom(view))
}

func (b *messageEventBridge) OnMessageRead(_ context.Context, projectID, messageID, readerID uuid.UUID) {
	if b == nil || b.pub == nil {
		return
	}
	b.pub.PublishReadUpdated(projectID, messageID, readerID)
}

func messageItemPayloadFrom(v usecase.MessageItemView) wshub.MessageItemPayload {
	out := wshub.MessageItemPayload{
		ID:         v.ID.String(),
		ProjectID:  v.ProjectID.String(),
		SenderID:   v.SenderID.String(),
		ReceiverID: v.ReceiverID.String(),
		Content:    v.Content,
		IsRead:     v.IsRead,
		CreatedAt:  v.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if v.Sender != nil {
		s := userPayloadFrom(*v.Sender)
		out.Sender = &s
	}
	if v.Receiver != nil {
		r := userPayloadFrom(*v.Receiver)
		out.Receiver = &r
	}
	return out
}

func userPayloadFrom(v usecase.UserView) wshub.UserPayload {
	return wshub.UserPayload{
		ID:        v.ID.String(),
		Email:     v.Email,
		Name:      v.Name,
		Role:      v.Role,
		IsActive:  v.IsActive(),
		AvatarURL: v.AvatarURL,
	}
}


const testJWTSecret = "01234567890123456789012345678901"

var (
	testGDB *gorm.DB
	testRDB *redis.Client
)

type seedOpts struct {
	Role          model.UserRole
	AdminRole     *model.AdminRole
	TOTPEnabled   bool
	AccountStatus model.AccountStatus
}

func TestMain(m *testing.M) {
	var (
		code int
		err  error
	)
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
		err = initTestInfra()
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration infra: %v\n", err)
			os.Exit(1)
		}
		defer closeTestInfra()
	}
	code = m.Run()
	os.Exit(code)
}

func initTestInfra() error {
	var (
		sqlDB *sql.DB
		err   error
	)
	err = db.Up(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	testGDB, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	sqlDB, err = testGDB.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Ping()
	if err != nil {
		return err
	}
	testRDB = db.OpenRedis()
	return nil
}

func closeTestInfra() {
	var sqlDB *sql.DB
	if testGDB != nil {
		sqlDB, _ = testGDB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		testGDB = nil
	}
	if testRDB != nil {
		_ = testRDB.Close()
		testRDB = nil
	}
}

func appOpts(stripeClient stripe.Client) *Options {
	return &Options{Stripe: stripeClient, DB: testGDB, Redis: testRDB}
}

func setupApp(t *testing.T) *echo.Echo {
	return setupAppWithOpts(t, nil)
}

func setupAppWithOpts(t *testing.T, opts *Options) *echo.Echo {
	var (
		e   *echo.Echo
		err error
	)
	t.Helper()
	if testGDB == nil {
		t.Skip("DATABASE_URL が未設定のためスキップ")
	}
	_ = os.Setenv("JWT_SECRET", testJWTSecret)
	if opts == nil {
		opts = appOpts(nil)
	}
	e, _, err = BuildHTTP(opts)
	if err != nil {
		t.Fatalf("BuildHTTP: %v", err)
	}
	return e
}

func setupPaymentApp(t *testing.T) (*echo.Echo, *stripe.StubClient) {
	var stub *stripe.StubClient
	t.Helper()
	stub = stripe.NewStubClient()
	return setupAppWithOpts(t, appOpts(stub)), stub
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testGDB == nil {
		t.Fatal("DATABASE_URL が未設定")
	}
	return testGDB
}

func seedUser(t *testing.T, gdb *gorm.DB, opts seedOpts) *model.User {
	var (
		hash string
		now  time.Time
		u    *model.User
		err  error
	)
	t.Helper()
	hash, err = usecase.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	now = time.Now().UTC()
	u = &model.User{
		ID:            uuid.New(),
		Email:         fmt.Sprintf("%s-%s@test.local", opts.Role, uuid.New()),
		PasswordHash:  hash,
		Role:          opts.Role,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusNotApplicable,
		AdminRole:     opts.AdminRole,
		TOTPEnabled:   opts.TOTPEnabled,
		TokenVersion:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if opts.AccountStatus != "" {
		u.AccountStatus = opts.AccountStatus
	}
	if opts.Role == model.UserRoleSeller {
		u.SellerStatus = model.SellerStatusActive
	}
	if err = gdb.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func seedProject(t *testing.T, gdb *gorm.DB, sellerID uuid.UUID, status model.ProjectStatus) *model.Project {
	var (
		now time.Time
		p   *model.Project
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &model.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        fmt.Sprintf("project-%s", uuid.New()),
		Description:  "integration test project",
		TargetAmount: 1000,
		DurationType: model.DurationUnlimited,
		StartDate:    now,
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = gdb.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	return p
}

func issueToken(u *model.User, twoFA bool) (string, error) {
	return usecase.NewTokenService([]byte(testJWTSecret), usecase.DefaultAuthTTL).Issue(usecase.AuthClaims{
		UserID: u.ID, Role: u.Role, TwoFAVerified: twoFA,
		AdminRole: u.AdminRole, TokenVersion: u.TokenVersion,
	})
}

func issueAdminToken(t *testing.T, gdb *gorm.DB, role model.AdminRole) string {
	var (
		u   *model.User
		tok string
		err error
	)
	t.Helper()
	u = seedUser(t, gdb, seedOpts{Role: model.UserRoleAdmin, AdminRole: &role, TOTPEnabled: true})
	tok, err = issueToken(u, true)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func httpGET(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPOST(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPUT(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpPATCH(e *echo.Echo, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func httpDELETE(e *echo.Echo, path, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func mediaUploadPOST(e *echo.Echo, token string, data []byte, filename string) *httptest.ResponseRecorder {
	var (
		body   bytes.Buffer
		writer *multipart.Writer
		part   io.Writer
		rec    *httptest.ResponseRecorder
		req    *http.Request
		err    error
	)
	writer = multipart.NewWriter(&body)
	part, err = writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	_, _ = part.Write(data)
	_ = writer.Close()
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/media/images", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
}

func expiredToken(userID uuid.UUID) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(), "role": string(model.UserRoleBuyer),
		"2fa_verified": false, "tv": float64(0),
		"exp": time.Now().Add(-time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte(testJWTSecret))
	if err != nil {
		return ""
	}
	return s
}
