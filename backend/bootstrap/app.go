package bootstrap

import (
	"log/slog"
	"os"

	"crowdfund-app/db"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Options はテスト・バッチ向けの依存注入。
type Options struct {
	Stripe stripe.Client
	DB     *gorm.DB
	Redis  *redis.Client
}

// Application は構築済み HTTP アプリケーション。
type Application struct {
	Echo   *echo.Echo
	Config AppConfig
	Addr   string
}

// BuildApplication は composition root として HTTP アプリを構築する。
func BuildApplication(opts *Options) (*Application, error) {
	var (
		err          error
		cfg          AppConfig
		gormDB       *gorm.DB
		rdb          *redis.Client
		stripeClient stripe.Client
		repos        Repositories
		ucs          UseCases
		ctls         Controllers
		mw           MiddlewareBundle
		origins      []string
		e            *echo.Echo
	)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err = LoadConfig()
	if err != nil {
		return nil, err
	}
	if opts != nil && opts.DB != nil {
		gormDB = opts.DB
	} else {
		gormDB, err = OpenDB()
		if err != nil {
			return nil, err
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
		stripeClient = StripeFromEnv()
	}

	repos = NewRepositories(gormDB, rdb, cfg)
	ucs = NewUseCases(repos, cfg, gormDB, stripeClient)
	origins = CORSOrigins()
	ctls = NewControllers(ucs, ucs.KYCAccess, gormDB, rdb, repos.WSHub, cfg, stripeClient, origins)
	mw = BuildMiddlewares(ucs.Auth, repos.Activity, ucs.RateLimit, cfg, origins)
	e = BuildRouter(ctls, ucs, mw, cfg)
	RegisterUploads(e, cfg.Storage.PublicRoot)

	return &Application{
		Echo:   e,
		Config: cfg,
		Addr:   cfg.ListenAddr(),
	}, nil
}

// BuildHTTP は後方互換のエイリアス（非推奨: BuildApplication を使用）。
func BuildHTTP(opts *Options) (*echo.Echo, string, error) {
	var (
		app *Application
		err error
	)
	app, err = BuildApplication(opts)
	if err != nil {
		return nil, "", err
	}
	return app.Echo, app.Addr, nil
}

// BatchApp はバッチジョブ用の最小配線。
type BatchApp struct {
	DB    *gorm.DB
	Batch *usecase.BatchUseCase
}

// OpenBatch はバッチ用ユースケースを構築する。
func OpenBatch(opts *Options) (*BatchApp, error) {
	var (
		err          error
		gormDB       *gorm.DB
		stripeClient stripe.Client
		repos        Repositories
		ucs          UseCases
		cfg          AppConfig
	)
	gormDB, err = OpenDB()
	if err != nil {
		return nil, err
	}
	stripeClient = StripeFromEnv()
	if opts != nil && opts.Stripe != nil {
		stripeClient = opts.Stripe
	}
	cfg, err = LoadConfig()
	if err != nil {
		return nil, err
	}
	repos = NewRepositories(gormDB, db.OpenRedis(), cfg)
	ucs = NewUseCases(repos, cfg, gormDB, stripeClient)
	return &BatchApp{DB: gormDB, Batch: ucs.Batch}, nil
}
