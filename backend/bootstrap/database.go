package bootstrap

import (
	"database/sql"

	"crowdfund-app/db"

	"gorm.io/gorm"
)

// OpenDB はマイグレーション適用後に DB 接続を開く。
func OpenDB() (*gorm.DB, error) {
	var (
		url    string
		gormDB *gorm.DB
		sqlDB  *sql.DB
		err    error
	)
	url = DatabaseURL()
	if err = db.Up(url); err != nil {
		return nil, err
	}
	gormDB, err = db.Open(url)
	if err != nil {
		return nil, err
	}
	sqlDB, err = gormDB.DB()
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}
	return gormDB, nil
}
