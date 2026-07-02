package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"crowdfund-app/bootstrap"
	"crowdfund-app/usecase"

	"github.com/joho/godotenv"
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
		app *bootstrap.BatchApp
		run *usecase.BatchRunView
		err error
	)
	ctx = context.Background()
	app, err = bootstrap.OpenBatch(nil)
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
