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

func isSupportedBatchJob(name string) bool {
	switch name {
	case entity.ApplyPlans, entity.ExpireSupporters:
		return true
	default:
		return false
	}
}
