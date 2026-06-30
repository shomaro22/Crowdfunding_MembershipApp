package controller

import (
	"context"
	"database/sql"

	"crowdfund-app/db"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type healthController struct {
	SQL *gorm.DB
	RDB *redis.Client
}

func NewHealthController(sql *gorm.DB, rdb *redis.Client) IHealthController {
	return &healthController{SQL: sql, RDB: rdb}
}

func (h *healthController) Health(c echo.Context) error {
	var (
		ctx  context.Context
		body map[string]string
	)
	ctx = reqCtx(c)
	body = map[string]string{
		"status": "ok",
		"db":     pingDB(ctx, h.SQL),
		"redis":  pingRedis(ctx, h.RDB),
	}
	return jsonOK(c, body)
}

func pingDB(ctx context.Context, gormDB *gorm.DB) string {
	var (
		sqlDB *sql.DB
		err   error
	)
	if gormDB == nil {
		return "skipped"
	}
	sqlDB, err = gormDB.DB()
	if err != nil {
		return "error"
	}
	if err = sqlDB.PingContext(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func pingRedis(ctx context.Context, rdb *redis.Client) string {
	var err error
	if rdb == nil {
		return "skipped"
	}
	if err = db.PingRedis(ctx, rdb); err != nil {
		return "error"
	}
	return "ok"
}
