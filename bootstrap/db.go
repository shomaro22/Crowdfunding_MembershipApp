package bootstrap

import (
	"database/sql"

	"crowdfund-app/db"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func OpenDB() (*gorm.DB, error) {
	var (
		gormDB *gorm.DB
		sqlDB  *sql.DB
		err    error
	)
	err = db.Up(DatabaseURL())
	if err != nil {
		return nil, err
	}
	gormDB, err = db.Open(DatabaseURL())
	if err != nil {
		return nil, err
	}
	sqlDB, err = gormDB.DB()
	if err != nil {
		return nil, err
	}
	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}
	return gormDB, nil
}

func OpenRedis() *redis.Client {
	return db.OpenRedis()
}
