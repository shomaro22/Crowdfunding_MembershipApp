package usecase

import (
	"context"
	"fmt"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BatchUseCase struct {
	DB         *gorm.DB
	Runs       repository.BatchRepo
	Supporters *SupporterUseCase
	WSTokens   repository.WSTokenRepo
}

func (b *BatchUseCase) Run(ctx context.Context, jobName, triggeredBy string, dryRun bool) (*BatchRunView, error) {
	var (
		run       *entity.BatchRun
		processed int
		failed    int
		runErr    error
		finished  time.Time
		err       error
	)
	if !isSupportedBatchJob(jobName) {
		return nil, ErrBadRequest
	}
	run = &entity.BatchRun{
		ID:          uuid.New(),
		JobName:     jobName,
		Status:      entity.BatchRunning,
		StartedAt:   time.Now().UTC(),
		TriggeredBy: triggeredBy,
		DryRun:      dryRun,
	}
	if err = b.Runs.Create(ctx, run); err != nil {
		return nil, err
	}
	switch jobName {
	case entity.ApplyPlans:
		processed, failed, runErr = b.Supporters.RunApplyPlans(ctx, dryRun)
	case entity.ExpireSupporters:
		processed, failed, runErr = b.Supporters.RunExpireSupporters(ctx, dryRun)
	case entity.PurgeTokens:
		processed, failed, runErr = b.runPurgeTokens(ctx, dryRun)
	default:
		runErr = ErrBadRequest
	}
	finished = time.Now().UTC()
	run.FinishedAt = &finished
	run.Processed = processed
	run.Failed = failed
	if runErr != nil {
		run.Status = entity.BatchFailed
		msg := runErr.Error()
		run.ErrorMessage = &msg
	} else {
		run.Status = entity.BatchSuccess
		if failed > 0 {
			msg := fmt.Sprintf("%d records failed", failed)
			run.ErrorMessage = &msg
		}
	}
	if err = b.Runs.Update(ctx, run); err != nil {
		return BatchRunViewFrom(run), err
	}
	return BatchRunViewFrom(run), runErr
}

func (b *BatchUseCase) runPurgeTokens(ctx context.Context, dryRun bool) (processed, failed int, err error) {
	var (
		n   int64
		now time.Time
	)
	if b == nil || b.WSTokens == nil {
		return 0, 0, ErrBadRequest
	}
	now = time.Now().UTC()
	if dryRun {
		return 0, 0, nil
	}
	n, err = b.WSTokens.PurgeBefore(ctx, now)
	if err != nil {
		return 0, 1, err
	}
	return int(n), 0, nil
}

func isSupportedBatchJob(name string) bool {
	switch name {
	case entity.ApplyPlans, entity.ExpireSupporters, entity.PurgeTokens:
		return true
	default:
		return false
	}
}
