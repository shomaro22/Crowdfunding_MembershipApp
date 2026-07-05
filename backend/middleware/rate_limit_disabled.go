package middleware

import (
	"os"
	"strings"
)

// RateLimitDisabled returns true when RATE_LIMIT_DISABLED is "1" or "true".
// Intended for CI k6 load tests only — never enable in production.
func RateLimitDisabled() bool {
	var v = strings.TrimSpace(os.Getenv("RATE_LIMIT_DISABLED"))
	return v == "1" || strings.EqualFold(v, "true")
}
