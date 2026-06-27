package usecase

import (
	"errors"
	"io"

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

func mapKYCStoreError(err error) error {
	switch {
	case errors.Is(err, repository.ErrBadMedia):
		return ErrUnsupportedMedia
	case errors.Is(err, repository.ErrTooLarge):
		return ErrFileTooLarge
	default:
		return err
	}
}
