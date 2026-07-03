package usecase

import (
	"context"
	"time"

	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func InvalidateUserSessions(
	ctx context.Context,
	txBeginner repository.TxBeginner,
	users repository.IUserRepo,
	refresh repository.IRefreshTokenRepo,
	userID uuid.UUID,
) error {
	var err error

	return txBeginner.WithTx(ctx, func(txCtx context.Context) error {
		if _, err = users.BumpTokenVer(txCtx, userID); err != nil {
			return err
		}
		if refresh == nil {
			return nil
		}
		return refresh.RevokeAllByUser(txCtx, userID, time.Now().UTC())
	})
}
