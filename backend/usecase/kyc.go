package usecase

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	
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
	return SignKYCAccess(k.SignSecret, storageKey, now)
}

func (k *KYCAccessUseCase) OpenWithToken(_ context.Context, token string) (io.ReadCloser, string, error) {
	var (
		key string
		f   *os.File
		ct  string
		err error
	)
	key, err = VerifyKYCAccess(k.SignSecret, token, time.Now().UTC())
	if err != nil {
		if errors.Is(err, ErrInvalidKYCToken) {
			return nil, "", ErrInvalidToken
		}
		return nil, "", err
	}
	f, ct, err = k.KYC.Open(key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	return f, ct, nil
}
