package bootstrap

import (
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// RegisterUploads は /uploads/* を PUBLIC_ROOT/uploads から配信する。
// echo.Static の prefix 結合の誤りを避け、パストラバーサルも防ぐ。
func RegisterUploads(e *echo.Echo, publicRoot string) {
	uploadsRoot := filepath.Clean(filepath.Join(publicRoot, "uploads"))
	e.GET("/uploads/*", func(c echo.Context) error {
		var (
			rel    string
			abs    string
			prefix string
		)
		rel = strings.TrimPrefix(c.Param("*"), "/")
		if rel == "" || strings.Contains(rel, "..") {
			return echo.ErrNotFound
		}
		abs = filepath.Clean(filepath.Join(uploadsRoot, rel))
		prefix = uploadsRoot + string(filepath.Separator)
		if abs != uploadsRoot && !strings.HasPrefix(abs, prefix) {
			return echo.ErrNotFound
		}
		return c.File(abs)
	})
}
