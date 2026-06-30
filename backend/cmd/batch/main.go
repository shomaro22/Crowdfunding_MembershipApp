package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"crowdfund-app/db"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"
	"crowdfund-app/usecase"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	var (
		dryRun bool
		args   []string
		job    string
		err    error
	)
	flag.BoolVar(&dryRun, "dry-run", false, "count targets without writing changes")
	flag.Parse()
	args = flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: batch [-dry-run] <apply_plans|expire_supporters|purge_tokens>\n")
		os.Exit(2)
	}
	job = args[0]
	loadDotEnv()
	err = runBatch(job, dryRun)
	if err != nil {
		log.Fatal(err)
	}
}

func runBatch(job string, dryRun bool) error {
	var (
		ctx context.Context
		app *batchApp
		run *usecase.BatchRunView
		err error
	)
	ctx = context.Background()
	app, err = openBatch()
	if err != nil {
		return err
	}
	run, err = app.Batch.Run(ctx, job, "cli", dryRun)
	if err != nil {
		return err
	}
	log.Printf("batch %s: status=%s processed=%d failed=%d dry_run=%v", run.JobName, run.Status, run.Processed, run.Failed, run.DryRun)
	if run.ErrorMessage != nil {
		log.Printf("batch note: %s", *run.ErrorMessage)
	}
	return nil
}

func loadDotEnv() {
	const envFile = ".env"
	if _, err := os.Stat(envFile); err != nil {
		return
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: load .env: %v", err)
	}
}

func databaseURL() string {
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	return u
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

func stripeFromEnv() stripe.Client {
	cfg, err := stripe.ConfigFromEnv()
	if err != nil {
		return nil
	}
	return stripe.New(cfg)
}

type batchApp struct {
	DB    *gorm.DB
	Batch *usecase.BatchUseCase
}

func openBatch() (*batchApp, error) {
	gormDB, err := openDB()
	if err != nil {
		return nil, err
	}
	batchUC, err := newBatchUsecase(gormDB, stripeFromEnv())
	if err != nil {
		return nil, err
	}
	return &batchApp{DB: gormDB, Batch: batchUC}, nil
}

func newBatchUsecase(gormDB *gorm.DB, stripeClient stripe.Client) (*usecase.BatchUseCase, error) {
	users := repository.NewUserRepo(gormDB)
	projects := repository.NewProjectRepo(gormDB)
	supporters := repository.NewSupporterRepo(gormDB)
	purchases := repository.NewPurchaseRepo(gormDB)
	connect := repository.NewConnectRepo(gormDB)
	activity := repository.NewActivityRepo(gormDB)
	batch := repository.NewBatchRepo(gormDB)
	wsTokens := repository.NewWSTokenRepo(gormDB)
	supportUC := usecase.NewSupporterUsecase(gormDB, users, projects, supporters, purchases, connect, activity, stripeClient)
	return usecase.NewBatchUsecase(gormDB, batch, supportUC, wsTokens), nil
}
