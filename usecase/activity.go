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

func (p *PaymentUseCase) logPurchaseCreated(ctx context.Context, purchase *entity.Purchase, meta RequestMeta) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	LogActivity(ctx, p.Activity, uid, entity.PurchaseCreated, "purchase", purchase.ID.String(), meta, purchaseActivityMeta(purchase))
}

func (p *PaymentUseCase) logPaymentCheckoutStarted(ctx context.Context, purchase *entity.Purchase, meta RequestMeta, flow string) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	fields := map[string]string{"flow": flow}
	if purchase != nil {
		fields["project_id"] = purchase.ProjectID.String()
	}
	LogActivity(ctx, p.Activity, uid, entity.PaymentCheckout, "purchase", purchase.ID.String(), meta, activityMeta(fields))
}

func (p *PaymentUseCase) logPurchaseCancelled(ctx context.Context, purchase *entity.Purchase, reason string, to entity.PurchaseStatus) {
	var uid *uuid.UUID
	if purchase != nil {
		uid = &purchase.BuyerID
	}
	LogActivity(ctx, p.Activity, uid, entity.PurchaseCancelled, "purchase", purchase.ID.String(), RequestMeta{}, activityMeta(map[string]string{
		"reason":    reason,
		"to_status": string(to),
	}))
}

func (p *PaymentUseCase) logPaymentWebhook(ctx context.Context, eventType, eventID string) {
	LogActivity(ctx, p.Activity, nil, entity.PaymentWebhook, "stripe_event", eventID, RequestMeta{}, activityMeta(map[string]string{
		"event_type": eventType,
	}))
}
