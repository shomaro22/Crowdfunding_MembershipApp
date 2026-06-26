package repository

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// WithTx runs fn inside a database transaction. Repositories resolve the active
// transaction via dbFromCtx when the context carries txKey.
func WithTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func dbFromCtx(ctx context.Context, db *gorm.DB) *gorm.DB {
	var (
		tx *gorm.DB
		ok bool
	)
	tx, ok = ctx.Value(txKey{}).(*gorm.DB)
	if ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
