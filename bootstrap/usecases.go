package bootstrap

import (
	"log/slog"

	"crowdfund-app/auth"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"gorm.io/gorm"
)

type UseCases struct {
	Auth      *usecase.AuthUseCase
	Admin     *usecase.AdminUseCase
	Project   *usecase.ProjectUseCase
	Connect   *usecase.ConnectUseCase
	Payment   *usecase.PaymentUseCase
	Supporter *usecase.SupporterUseCase
	Purchase  *usecase.PurchaseUseCase
	RateLimit *usecase.RateLimitUseCase
	KYCAccess *usecase.KYCAccessUseCase
}

func NewUseCases(
	db *gorm.DB,
	repos Repositories,
	redisRepos RedisRepositories,
	kyc *repository.LocalKYCStore,
	stripeClient stripe.Client,
	cfg Config,
) UseCases {
	var (
		tokenSvc  = auth.NewTokenService([]byte(cfg.JWTSecret), auth.DefaultAuthTTL)
		authUC    *usecase.AuthUseCase
		connectUC *usecase.ConnectUseCase
		supportUC *usecase.SupporterUseCase
		google    *auth.GoogleOAuth
		err       error
	)

	authUC = &usecase.AuthUseCase{
		Users:         repos.Users,
		SellerRegs:    repos.SellerRegs,
		RefreshTokens: repos.RefreshTokens,
		ResetToks:     repos.ResetToks,
		Activity:      repos.Activity,
		OAuth:         repos.OAuth,
		OAuthCache:    redisRepos.OAuthCache,
		TOTPBackup:    repos.TOTPBackup,
		Tokens:        tokenSvc,
		KYCStore:      kyc,
		TOTPKey:       []byte(cfg.JWTSecret),
		TOTPIssuer:    cfg.TOTPIssuer,
		DB:            db,
		AppPublicURL:  cfg.AppPublicURL,
	}
	google, err = auth.GoogleFromEnv(auth.NewStateBridge(redisRepos.OAuthCache))
	if err == nil {
		authUC.Google = google
	} else {
		slog.Info("google oauth disabled", "reason", err.Error())
	}

	connectUC = &usecase.ConnectUseCase{
		Users:   repos.Users,
		Connect: repos.Connect,
		Stripe:  stripeClient,
	}
	supportUC = NewSupporterUseCase(db, repos, stripeClient)
	paymentUC := &usecase.PaymentUseCase{
		DB:              db,
		Users:           repos.Users,
		Projects:        repos.Projects,
		Purchases:       repos.Purchases,
		Supporters:      repos.Supporters,
		Connect:         repos.Connect,
		Webhooks:        repos.Webhooks,
		Activity:        repos.Activity,
		Stripe:          stripeClient,
		ConnectSync:     connectUC,
		SupporterViewer: supportUC,
	}

	return UseCases{
		Auth: authUC,
		Admin: &usecase.AdminUseCase{
			Users:           repos.Users,
			SellerRegs:      repos.SellerRegs,
			Projects:        repos.Projects,
			Audit:           repos.Audit,
			ReviewDecisions: repos.ReviewDecisions,
			RefreshTokens:   repos.RefreshTokens,
			TOTPBackup:      repos.TOTPBackup,
			KYC:             kyc,
			DB:              db,
			APIBase:         cfg.APIPublicURL,
			SignSecret:      []byte(cfg.JWTSecret),
		},
		Project:   &usecase.ProjectUseCase{Projects: repos.Projects, Users: repos.Users, DB: db},
		Connect:   connectUC,
		Payment:   paymentUC,
		Supporter: supportUC,
		Purchase: &usecase.PurchaseUseCase{
			Users:     repos.Users,
			Projects:  repos.Projects,
			Purchases: repos.Purchases,
		},
		RateLimit: &usecase.RateLimitUseCase{Repo: redisRepos.RateLimit},
		KYCAccess: &usecase.KYCAccessUseCase{KYC: kyc, SignSecret: []byte(cfg.JWTSecret)},
	}
}

func NewSupporterUseCase(db *gorm.DB, repos Repositories, stripeClient stripe.Client) *usecase.SupporterUseCase {
	return &usecase.SupporterUseCase{
		DB:         db,
		Users:      repos.Users,
		Projects:   repos.Projects,
		Supporters: repos.Supporters,
		Purchases:  repos.Purchases,
		Connect:    repos.Connect,
		Activity:   repos.Activity,
		Stripe:     stripeClient,
	}
}

func NewBatchUseCase(db *gorm.DB, repos Repositories, stripeClient stripe.Client) *usecase.BatchUseCase {
	return &usecase.BatchUseCase{
		DB:         db,
		Runs:       repos.Batch,
		Supporters: NewSupporterUseCase(db, repos, stripeClient),
	}
}
