package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const MaxKYCBytes = 5 << 20 // 5 MiB

var (
	ErrBadMedia = errors.New("unsupported media type")
	ErrTooLarge = errors.New("file too large")
	ErrBadToken = errors.New("invalid kyc access token")
)

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
	Open(key string) (*os.File, string, error)
}

type LocalKYCStore struct {
	Root string
}

func NewLocalKYC(root string) *LocalKYCStore {
	return &LocalKYCStore{Root: root}
}

func (s *LocalKYCStore) Save(_ context.Context, userID uuid.UUID, side string, img KYCImage) (string, error) {
	var (
		ext     string
		ok      bool
		dir     string
		name    string
		absPath string
		f       *os.File
		n       int64
		key     string
		err     error
	)
	ext, ok = allowedKYCTypes[strings.ToLower(strings.TrimSpace(img.ContentType))]
	if !ok {
		return "", ErrBadMedia
	}
	if img.Size <= 0 || img.Size > MaxKYCBytes {
		return "", ErrTooLarge
	}

	dir = filepath.Join(s.Root, "kyc", userID.String())
	err = os.MkdirAll(dir, 0o700)
	if err != nil {
		return "", err
	}

	name = fmt.Sprintf("%s_%s%s", uuid.NewString(), side, ext)
	absPath = filepath.Join(dir, name)
	f, err = os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	n, err = io.Copy(f, io.LimitReader(img.Reader, MaxKYCBytes+1))
	if err != nil {
		_ = os.Remove(absPath)
		return "", err
	}
	if n > MaxKYCBytes {
		_ = os.Remove(absPath)
		return "", ErrTooLarge
	}

	key = filepath.ToSlash(filepath.Join("kyc", userID.String(), name))
	return key, nil
}

func (s *LocalKYCStore) Remove(_ context.Context, keys ...string) error {
	var (
		key string
		abs string
	)
	for _, key = range keys {
		key = filepath.Clean(key)
		if strings.Contains(key, "..") {
			continue
		}
		abs = filepath.Join(s.Root, key)
		_ = os.Remove(abs)
	}
	return nil
}

func (s *LocalKYCStore) Open(key string) (*os.File, string, error) {
	var (
		abs string
		f   *os.File
		ct  string
		err error
	)
	key = filepath.Clean(key)
	if strings.Contains(key, "..") {
		return nil, "", ErrBadMedia
	}
	abs = filepath.Join(s.Root, key)
	f, err = os.Open(abs)
	if err != nil {
		return nil, "", err
	}
	ct = "application/octet-stream"
	switch filepath.Ext(abs) {
	case ".jpg", ".jpeg":
		ct = "image/jpeg"
	case ".png":
		ct = "image/png"
	case ".webp":
		ct = "image/webp"
	}
	return f, ct, nil
}
