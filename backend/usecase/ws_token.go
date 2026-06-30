package usecase

import (
	"context"
	"strings"
	"time"

	
	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

const wsTokenTTL = 60 * time.Second

type WSTokenUseCase struct {
	Tokens   repository.IWSTokenRepo
	Users    repository.IUserRepo
	Activity repository.IActivityRepo
}

func (w *WSTokenUseCase) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	var (
		u     model.User
		plain string
		now   time.Time
		row   *model.WSToken
		err   error
	)
	u, err = w.Users.GetID(ctx, userID)
	if err != nil {
		return "", mapRepoNotFoundAs(err, ErrForbidden)
	}
	if !u.IsActive() {
		return "", ErrForbidden
	}
	plain, err = OpaqueHex()
	if err != nil {
		return "", err
	}
	now = time.Now().UTC()
	row = &model.WSToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: OpaqueHash(plain),
		ExpiresAt: now.Add(wsTokenTTL),
		CreatedAt: now,
	}
	if err = w.Tokens.Create(ctx, row); err != nil {
		return "", err
	}
	return plain, nil
}

func (w *WSTokenUseCase) Consume(ctx context.Context, plain string, meta RequestMeta) (uuid.UUID, error) {
	var (
		row   *model.WSToken
		u     model.User
		now   time.Time
		err   error
	)
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return uuid.Nil, ErrInvalidToken
	}
	now = time.Now().UTC()
	row, err = w.Tokens.ConsumeValidHash(ctx, OpaqueHash(plain), now)
	if err != nil {
		return uuid.Nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = w.Users.GetID(ctx, row.UserID)
	if err != nil {
		return uuid.Nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.IsActive() {
		return uuid.Nil, ErrInvalidToken
	}
	LogActivity(ctx, w.Activity, &row.UserID, model.SecWSConnect, "ws_token", row.ID.String(), meta, nil)
	return row.UserID, nil
}
