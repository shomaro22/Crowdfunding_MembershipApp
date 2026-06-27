package bootstrap

import (
	"crowdfund-app/repository"
	"crowdfund-app/stripe"
	"crowdfund-app/wshub"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	DB         *gorm.DB
	RDB        *redis.Client
	Repos      Repositories
	RedisRepos RedisRepositories
	KYC        *repository.LocalKYCStore
	Stripe     stripe.Client
	Hub        *wshub.ConnectionManager
	UseCases   UseCases
	Config     Config
}

func OpenApp(opts *Options) (*App, error) {
	var (
		cfg        Config
		gormDB     *gorm.DB
		rdb        *redis.Client
		repos      Repositories
		redisRepos RedisRepositories
		kyc        *repository.LocalKYCStore
		stripeClient stripe.Client
		hub        *wshub.ConnectionManager
		useCases   UseCases
		app        *App
		err        error
	)
	cfg, err = LoadConfig()
	if err != nil {
		return nil, err
	}
	gormDB, err = OpenDB()
	if err != nil {
		return nil, err
	}
	rdb = OpenRedis()
	repos = NewRepositories(gormDB)
	redisRepos = NewRedisRepositories(rdb)
	kyc = repository.NewLocalKYC(cfg.PrivateRoot)
	stripeClient = stripeClientFrom(opts)
	hub = wshub.NewConnectionManager()
	useCases = NewUseCases(gormDB, repos, redisRepos, kyc, stripeClient, cfg)
	app = &App{
		DB:         gormDB,
		RDB:        rdb,
		Repos:      repos,
		RedisRepos: redisRepos,
		KYC:        kyc,
		Stripe:     stripeClient,
		Hub:        hub,
		UseCases:   useCases,
		Config:     cfg,
	}
	app.WireRealtimeMessaging()
	return app, nil
}

func (a *App) WireRealtimeMessaging() {
	if a == nil {
		return
	}
	wireMessageEvents(a.UseCases.Message, a.Hub)
}

func stripeClientFrom(opts *Options) stripe.Client {
	if opts != nil && opts.Stripe != nil {
		return opts.Stripe
	}
	return StripeFromEnv()
}
