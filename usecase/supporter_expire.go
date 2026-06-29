package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
)

func isExpirableStatus(status entity.SupporterStatus) bool {
	switch status {
	case entity.SupporterStatusActive, entity.SupporterStatusPendingRenewal, entity.SupporterStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *SupporterUseCase) RunExpireSupporters(ctx context.Context, dryRun bool) (processed, failed int, err error) {
	var (
		rows []entity.ProjectSupporter
		now  time.Time
		i    int
	)
	now = time.Now().UTC()
	rows, err = s.Supporters.ListExpirable(ctx, now)
	if err != nil {
		return 0, 0, err
	}
	if dryRun {
		return len(rows), 0, nil
	}
	for i = range rows {
		if err = s.expireSupporter(ctx, &rows[i], now); err != nil {
			failed++
			continue
		}
		processed++
	}
	return processed, failed, nil
}

func (s *SupporterUseCase) expireSupporter(ctx context.Context, supporter *entity.ProjectSupporter, now time.Time) error {
	var (
		row *entity.ProjectSupporter
		err error
	)
	if supporter == nil {
		return ErrNotFound
	}
	row, err = s.Supporters.FindID(ctx, supporter.ID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	if row.Status == entity.SupporterStatusExpired {
		return nil
	}
	if !now.After(row.CurrentPeriodEnd) || !isExpirableStatus(row.Status) {
		return nil
	}
	row.Status = entity.SupporterStatusExpired
	row.UpdatedAt = now
	return s.Supporters.Update(ctx, row)
}
