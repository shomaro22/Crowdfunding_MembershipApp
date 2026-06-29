package usecase

import (
	"context"
	"io"
	"strings"

	"crowdfund-app/repository"

	"github.com/google/uuid"
)

type MediaUploadIn struct {
	Reader      io.Reader
	ContentType string
	Size        int64
}

type MediaUseCase struct {
	Store        repository.IMediaStore
	APIPublicURL string
}

func (m *MediaUseCase) UploadImage(ctx context.Context, userID uuid.UUID, in MediaUploadIn) (string, error) {
	var (
		path string
		base string
		err  error
	)
	if m == nil || m.Store == nil {
		return "", ErrBadRequest
	}
	if in.Reader == nil || in.Size <= 0 {
		return "", ErrBadRequest
	}
	path, err = m.Store.Save(ctx, userID, repository.MediaImage{
		Reader:      in.Reader,
		ContentType: in.ContentType,
		Size:        in.Size,
	})
	if err != nil {
		return "", mapMediaStoreError(err)
	}
	base = strings.TrimRight(strings.TrimSpace(m.APIPublicURL), "/")
	if base == "" {
		return "", ErrBadRequest
	}
	return base + "/" + path, nil
}
