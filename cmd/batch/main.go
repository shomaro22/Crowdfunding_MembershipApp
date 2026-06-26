package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"crowdfund-app/db"
	"crowdfund-app/entity"
	"crowdfund-app/repository"
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
		fmt.Fprintf(os.Stderr, "usage: batch [-dry-run] <apply_plans|expire_supporters>\n")
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
		ctx         context.Context
		databaseURL string
		gormDB      *gorm.DB
		batchUC     *usecase.BatchUseCase
		run         *entity.BatchRun
		err         error
	)
	ctx = context.Background()
	databaseURL = os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/crowdfund?sslmode=disable"
	}
	err = db.Up(databaseURL)
	if err != nil {
		return err
	}
	gormDB, err = db.Open(databaseURL)
	if err != nil {
		return err
	}
	batchUC = buildBatchUseCase(gormDB)
	run, err = batchUC.Run(ctx, job, "cli", dryRun)
	if err != nil {
		return err
	}
	log.Printf("batch %s: status=%s processed=%d failed=%d dry_run=%v", run.JobName, run.Status, run.Processed, run.Failed, run.DryRun)
	if run.ErrorMessage != nil {
		log.Printf("batch note: %s", *run.ErrorMessage)
	}
	return nil
}

func buildBatchUseCase(gormDB *gorm.DB) *usecase.BatchUseCase {
	var (
		supporterRepo repository.SupporterRepo
		batchRepo     repository.BatchRepo
		supporterUC   *usecase.SupporterUseCase
	)
	supporterRepo = repository.NewSupporterRepo(gormDB)
	batchRepo = repository.NewBatchRepo(gormDB)
	supporterUC = &usecase.SupporterUseCase{
		DB:         gormDB,
		Supporters: supporterRepo,
	}
	return &usecase.BatchUseCase{
		DB:         gormDB,
		Runs:       batchRepo,
		Supporters: supporterUC,
	}
}

func loadDotEnv() {
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
