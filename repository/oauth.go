package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OAuthRepo interface {
	Create(ctx context.Context, row *entity.OAuthAccount) error
	FindSub(ctx context.Context, provider entity.OAuthProvider, sub string) (*entity.OAuthAccount, error)
	FindUserID(ctx context.Context, userID uuid.UUID, provider entity.OAuthProvider) (*entity.OAuthAccount, error)
}

type oauthRepo struct {
	db *gorm.DB
}

func NewOAuthRepo(db *gorm.DB) OAuthRepo {
	return &oauthRepo{db: db}
}

func (r *oauthRepo) Create(ctx context.Context, row *entity.OAuthAccount) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}

func (r *oauthRepo) FindSub(ctx context.Context, provider entity.OAuthProvider, sub string) (*entity.OAuthAccount, error) {
	var (
		m   entity.OAuthAccount
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("provider = ? AND provider_sub = ?", provider, sub).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *oauthRepo) FindUserID(ctx context.Context, userID uuid.UUID, provider entity.OAuthProvider) (*entity.OAuthAccount, error) {
	var (
		m   entity.OAuthAccount
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("user_id = ? AND provider = ?", userID, provider).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
