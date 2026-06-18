package controller

import (
	"context"
	"net/http"

	"crowdfund-app/db"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthController struct {
	SQL *gorm.DB
	RDB *redis.Client
}

func (h *HealthController) Health(c echo.Context) error {
	ctx := c.Request().Context()
	body := map[string]string{
		"status": "ok",
		"db":     pingDB(ctx, h.SQL),
		"redis":  pingRedis(ctx, h.RDB),
	}
	return c.JSON(http.StatusOK, body)
}

func pingDB(ctx context.Context, gormDB *gorm.DB) string {
	if gormDB == nil {
		return "skipped"
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return "error"
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func pingRedis(ctx context.Context, rdb *redis.Client) string {
	if rdb == nil {
		return "skipped"
	}
	if err := db.PingRedis(ctx, rdb); err != nil {
		return "error"
	}
	return "ok"
}
