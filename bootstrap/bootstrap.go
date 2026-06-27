// Package bootstrap wires shared dependencies for HTTP and batch entrypoints.
package bootstrap

import (
	"log"
	"log/slog"
	"os"

	"crowdfund-app/repository"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func LoadDotEnv() {
	const envFile = ".env"
	var err error
	_, err = os.Stat(envFile)
	if err != nil {
		return
	}
	err = godotenv.Load(envFile)
	if err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}

func DatabaseURL() string {
	var u string
	u = os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
}

type Repositories struct {
	Users      repository.UserRepo
	Projects   repository.ProjectRepo
	Supporters repository.SupporterRepo
	Purchases  repository.PurchaseRepo
	Connect    repository.ConnectRepo
	Activity   repository.ActivityRepo
	Batch      repository.BatchRepo
	Webhooks   repository.WebhookEventRepo
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		Users:      repository.NewUserRepo(db),
		Projects:   repository.NewProjectRepo(db),
		Supporters: repository.NewSupporterRepo(db),
		Purchases:  repository.NewPurchaseRepo(db),
		Connect:    repository.NewConnectRepo(db),
		Activity:   repository.NewActivityRepo(db),
		Batch:      repository.NewBatchRepo(db),
		Webhooks:   repository.NewWebhookEventRepo(db),
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

func StripeFromEnv() stripe.Client {
	var (
		cfg stripe.Config
		err error
	)
	cfg, err = stripe.ConfigFromEnv()
	if err != nil {
		slog.Info("stripe disabled", "reason", err.Error())
		return nil
	}
	return stripe.New(cfg)
}
