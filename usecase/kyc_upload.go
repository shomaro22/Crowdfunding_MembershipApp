package usecase

import (
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
	switch err {
	case repository.ErrBadMedia:
		return ErrUnsupportedMedia
	case repository.ErrTooLarge:
		return ErrFileTooLarge
	default:
		return err
	}
}
