package repository

import (
	"context"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

type AuditRepo interface {
	Create(ctx context.Context, row *entity.AuditLog) error
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, row *entity.AuditLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}
