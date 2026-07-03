package bootstrap

import (
	"net/url"
	"strings"
)

// NormalizeDatabaseURL は Neon 等の接続文字列を migrate / gorm 向けに整える。
func NormalizeDatabaseURL(raw string) string {
	var (
		u   *url.URL
		err error
		q   url.Values
	)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "postgresql://") {
		raw = "postgres://" + strings.TrimPrefix(raw, "postgresql://")
	}
	u, err = url.Parse(raw)
	if err != nil {
		return raw
	}
	q = u.Query()
	q.Del("channel_binding")
	u.RawQuery = q.Encode()
	return u.String()
}
