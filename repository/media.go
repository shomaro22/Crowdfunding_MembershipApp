package repository

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const MaxMediaBytes = 10 << 20 // 10 MiB

var allowedMediaTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type MediaImage struct {
	Reader      io.Reader
	ContentType string
	Size        int64
}

type IMediaStore interface {
	Save(ctx context.Context, userID uuid.UUID, img MediaImage) (publicPath string, err error)
}

type LocalMediaStore struct {
	Root string
}

func NewLocalMedia(root string) *LocalMediaStore {
	return &LocalMediaStore{Root: root}
}

func (s *LocalMediaStore) Save(_ context.Context, userID uuid.UUID, img MediaImage) (string, error) {
	var (
		ext     string
		ok      bool
		dir     string
		name    string
		absPath string
		f       *os.File
		n       int64
		err     error
	)
	ext, ok = allowedMediaTypes[strings.ToLower(strings.TrimSpace(img.ContentType))]
	if !ok {
		return "", ErrBadMedia
	}
	if img.Size <= 0 || img.Size > MaxMediaBytes {
		return "", ErrTooLarge
	}

	dir = filepath.Join(s.Root, "uploads", userID.String())
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return "", err
	}

	name = uuid.NewString() + ext
	absPath = filepath.Join(dir, name)
	f, err = os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	n, err = io.Copy(f, io.LimitReader(img.Reader, MaxMediaBytes+1))
	if err != nil {
		_ = os.Remove(absPath)
		return "", err
	}
	if n > MaxMediaBytes {
		_ = os.Remove(absPath)
		return "", ErrTooLarge
	}

	return fmt.Sprintf("uploads/%s/%s", userID.String(), name), nil
}
