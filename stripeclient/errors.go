package stripeclient

import "errors"

var (
	ErrNotConfigured      = errors.New("stripe is not configured")
	ErrWebhookNotConfigured = errors.New("stripe webhook secret is not configured")
)
