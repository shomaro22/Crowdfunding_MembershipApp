package auth

import (
	"net/url"
	"strings"
)

func PasswordResetURL(appPublicURL, plainToken string) string {
	var base string
	base = strings.TrimRight(strings.TrimSpace(appPublicURL), "/")
	return base + "/reset-password?token=" + url.QueryEscape(plainToken)
}
