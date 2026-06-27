package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"crowdfund-app/bootstrap"
	"crowdfund-app/entity"
	"crowdfund-app/usecase"

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
	bootstrap.LoadDotEnv()
	err = runBatch(job, dryRun)
	if err != nil {
		log.Fatal(err)
	}
}

func runBatch(job string, dryRun bool) error {
	var (
		ctx     context.Context
		gormDB  *gorm.DB
		batchUC *usecase.BatchUseCase
		run     *entity.BatchRun
		err     error
	)
	ctx = context.Background()
	gormDB, err = bootstrap.OpenDB()
	if err != nil {
		return err
	}
	repos := bootstrap.NewRepositories(gormDB)
	batchUC = bootstrap.NewBatchUseCase(gormDB, repos, bootstrap.StripeFromEnv())
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
