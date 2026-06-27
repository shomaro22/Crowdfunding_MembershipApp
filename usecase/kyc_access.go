package usecase

import (
	"context"
	"errors"
	"os"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/repository"
)

type KYCAccessUseCase struct {
	KYC        repository.KYCStore
	SignSecret []byte
}

func (k *KYCAccessUseCase) SignURL(apiBase, storageKey string, now time.Time) string {
	return auth.KYCAccessURL(apiBase, auth.SignKYCAccess(k.SignSecret, storageKey, now))
}

func (k *KYCAccessUseCase) OpenWithToken(_ context.Context, token string) (*os.File, string, error) {
	var (
		key string
		err error
	)
	key, err = auth.VerifyKYCAccess(k.SignSecret, token, time.Now().UTC())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidKYCToken) {
			return nil, "", ErrInvalidToken
		}
		return nil, "", err
	}
	return k.KYC.Open(key)
}
