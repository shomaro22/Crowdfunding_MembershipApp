package storage

import "errors"

var (
	ErrBadMedia = errors.New("unsupported media type")
	ErrTooLarge = errors.New("file too large")
	ErrBadToken = errors.New("invalid kyc access token")
)
