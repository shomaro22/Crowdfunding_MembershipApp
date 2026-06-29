package usecase

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/repository"
)

type KYCUpload struct {
	Reader      io.Reader
	ContentType string
	Size        int64
}

func (u KYCUpload) toStoreImage() repository.KYCImage {
	return repository.KYCImage{
		Reader:      u.Reader,
		ContentType: u.ContentType,
		Size:        u.Size,
	}
}

type KYCAccessUseCase struct {
	KYC        repository.IKYCStore
	SignSecret []byte
}

func (k *KYCAccessUseCase) SignToken(storageKey string, now time.Time) string {
	return auth.SignKYCAccess(k.SignSecret, storageKey, now)
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
