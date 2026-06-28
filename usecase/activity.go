package usecase

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

func LogActivity(
	ctx context.Context,
	repo repository.ActivityRepo,
	userID *uuid.UUID,
	action entity.ActivityAction,
	resource, resourceID string,
	meta RequestMeta,
	extra entity.JSONMetadata,
) {
	var (
		res   *string
		rid   *string
		ip    *string
		ua    *string
		reqID *string
	)
	if repo == nil {
		return
	}
	if resource != "" {
		res = &resource
	}
	if resourceID != "" {
		rid = &resourceID
	}
	if meta.IPAddress != "" {
		ip = &meta.IPAddress
	}
	if meta.UserAgent != "" {
		ua = &meta.UserAgent
	}
	if meta.RequestID != "" {
		reqID = &meta.RequestID
	}
	_ = repo.Create(ctx, &entity.ActivityLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     action,
		Resource:   res,
		ResourceID: rid,
		IPAddress:  ip,
		UserAgent:  ua,
		RequestID:  reqID,
		Metadata:   extra,
		CreatedAt:  time.Now().UTC(),
	})
}

func activityMeta(fields map[string]string) entity.JSONMetadata {
	var (
		b   []byte
		err error
	)
	if len(fields) == 0 {
		return nil
	}
	b, err = json.Marshal(fields)
	if err != nil {
		return nil
	}
	return entity.JSONMetadata(b)
}

func ActivityMeta(fields map[string]string) entity.JSONMetadata {
	return activityMeta(fields)
}

func LogRateLimitDenied(
	ctx context.Context,
	repo repository.ActivityRepo,
	userID *uuid.UUID,
	policy, ruleKey string,
	meta RequestMeta,
) {
	LogActivity(ctx, repo, userID, entity.SecRateLimit, "rate_limit", policy, meta, ActivityMeta(map[string]string{
		"policy": policy,
		"key":    ruleKey,
	}))
}

func purchaseActivityMeta(purchase *entity.Purchase) entity.JSONMetadata {
	if purchase == nil {
		return nil
	}
	return activityMeta(map[string]string{
		"project_id":     purchase.ProjectID.String(),
		"payment_method": string(purchase.PaymentMethod),
		"total_price":    strconv.Itoa(purchase.TotalPrice),
	})
}
