package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const MaxKYCBytes = 5 << 20 // 5 MiB

var allowedKYCTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type KYCImage struct {
	Reader      io.Reader
	ContentType string
	Size        int64
}

type KYCStore interface {
	Save(ctx context.Context, userID uuid.UUID, side string, img KYCImage) (key string, err error)
	Remove(ctx context.Context, keys ...string) error
}

type LocalKYCStore struct {
	Root string
}

func NewLocalKYC(root string) *LocalKYCStore {
	return &LocalKYCStore{Root: root}
}

func (s *LocalKYCStore) Save(_ context.Context, userID uuid.UUID, side string, img KYCImage) (string, error) {
	ext, ok := allowedKYCTypes[strings.ToLower(strings.TrimSpace(img.ContentType))]
	if !ok {
		return "", ErrBadMedia
	}
	if img.Size <= 0 || img.Size > MaxKYCBytes {
		return "", ErrTooLarge
	}

	dir := filepath.Join(s.Root, "kyc", userID.String())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s_%s%s", uuid.NewString(), side, ext)
	absPath := filepath.Join(dir, name)
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	n, err := io.Copy(f, io.LimitReader(img.Reader, MaxKYCBytes+1))
	if err != nil {
		_ = os.Remove(absPath)
		return "", err
	}
	if n > MaxKYCBytes {
		_ = os.Remove(absPath)
		return "", ErrTooLarge
	}

	key := filepath.ToSlash(filepath.Join("kyc", userID.String(), name))
	return key, nil
}

func (s *LocalKYCStore) Remove(_ context.Context, keys ...string) error {
	for _, key := range keys {
		key = filepath.Clean(key)
		if strings.Contains(key, "..") {
			continue
		}
		abs := filepath.Join(s.Root, key)
		_ = os.Remove(abs)
	}
	return nil
}
