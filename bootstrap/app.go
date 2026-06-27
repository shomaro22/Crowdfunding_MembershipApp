package bootstrap

import (
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

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
	UseCases   UseCases
	Config     Config
}

func OpenApp(opts *Options) (*App, error) {
	var (
		cfg      Config
		gormDB   *gorm.DB
		rdb      *redis.Client
		repos    Repositories
		redisRepos RedisRepositories
		kyc      *repository.LocalKYCStore
		stripeClient stripe.Client
		err      error
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
	return &App{
		DB:         gormDB,
		RDB:        rdb,
		Repos:      repos,
		RedisRepos: redisRepos,
		KYC:        kyc,
		Stripe:     stripeClient,
		UseCases:   NewUseCases(gormDB, repos, redisRepos, kyc, stripeClient, cfg),
		Config:     cfg,
	}, nil
}

func stripeClientFrom(opts *Options) stripe.Client {
	if opts != nil && opts.Stripe != nil {
		return opts.Stripe
	}
	return StripeFromEnv()
}
