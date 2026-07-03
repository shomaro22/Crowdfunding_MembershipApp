package repository

import (
	"context"

	"gorm.io/gorm"
)

// TxBeginner は usecase 層から *gorm.DB を隠蔽するトランザクション境界。
type TxBeginner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type gormTxBeginner struct {
	db *gorm.DB
}

func NewTxBeginner(db *gorm.DB) TxBeginner {
	return &gormTxBeginner{db: db}
}

func (g *gormTxBeginner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return WithTx(ctx, g.db, fn)
}
