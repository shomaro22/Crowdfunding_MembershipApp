package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

const wsTokenTTL = 60 * time.Second

type WSTokenUseCase struct {
	Tokens   repository.WSTokenRepo
	Users    repository.UserRepo
	Activity repository.ActivityRepo
}

func (w *WSTokenUseCase) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	var (
		u     *entity.User
		plain string
		now   time.Time
		row   *entity.WSToken
		err   error
	)
	u, err = w.Users.FindID(ctx, userID)
	if err != nil {
		return "", mapRepoNotFoundAs(err, ErrForbidden)
	}
	if !u.IsActive() {
		return "", ErrForbidden
	}
	plain, err = auth.OpaqueHex()
	if err != nil {
		return "", err
	}
	now = time.Now().UTC()
	row = &entity.WSToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: auth.OpaqueHash(plain),
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
		row   *entity.WSToken
		u     *entity.User
		now   time.Time
		err   error
	)
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return uuid.Nil, ErrInvalidToken
	}
	now = time.Now().UTC()
	row, err = w.Tokens.ConsumeValidHash(ctx, auth.OpaqueHash(plain), now)
	if err != nil {
		return uuid.Nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	u, err = w.Users.FindID(ctx, row.UserID)
	if err != nil {
		return uuid.Nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.IsActive() {
		return uuid.Nil, ErrInvalidToken
	}
	LogActivity(ctx, w.Activity, &row.UserID, entity.SecWSConnect, "ws_token", row.ID.String(), meta, nil)
	return row.UserID, nil
}
