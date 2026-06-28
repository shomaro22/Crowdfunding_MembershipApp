package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func WriteAudit(
	ctx context.Context,
	repo repository.AuditRepo,
	adminID uuid.UUID,
	action, resource, resourceID string,
	reason, details *string,
) {
	if repo == nil {
		return
	}
	_ = repo.Create(ctx, &entity.AuditLog{
		ID:         uuid.New(),
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Reason:     reason,
		Details:    details,
		CreatedAt:  time.Now().UTC(),
	})
}
