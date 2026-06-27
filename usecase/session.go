package usecase

import (
	"context"
	"time"

	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func InvalidateUserSessions(
	ctx context.Context,
	db *gorm.DB,
	users repository.UserRepo,
	refresh repository.RefreshTokenRepo,
	userID uuid.UUID,
) error {
	var err error

	return repository.WithTx(ctx, db, func(txCtx context.Context) error {
		if _, err = users.BumpTokenVer(txCtx, userID); err != nil {
			return err
		}
		if refresh == nil {
			return nil
		}
		return refresh.RevokeAllByUser(txCtx, userID, time.Now().UTC())
	})
}
