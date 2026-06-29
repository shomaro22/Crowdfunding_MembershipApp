package bootstrap

import (
	"context"
	"time"

	"crowdfund-app/usecase"
	"crowdfund-app/wshub"

	"github.com/google/uuid"
)

// messageEventBridge implements usecase.MessageEventHook without wshub importing usecase.
type messageEventBridge struct {
	pub *wshub.MessagePublisher
}

func newMessageEventBridge(hub *wshub.ConnectionManager) *messageEventBridge {
	return &messageEventBridge{pub: wshub.NewMessagePublisher(hub)}
}

func (b *messageEventBridge) OnMessageCreated(_ context.Context, view usecase.MessageItemView) {
	if b == nil || b.pub == nil {
		return
	}
	b.pub.PublishMessageCreated(view.ProjectID, messageItemPayloadFrom(view))
}

func (b *messageEventBridge) OnMessageRead(_ context.Context, projectID, messageID, readerID uuid.UUID) {
	if b == nil || b.pub == nil {
		return
	}
	b.pub.PublishReadUpdated(projectID, messageID, readerID)
}

func messageItemPayloadFrom(v usecase.MessageItemView) wshub.MessageItemPayload {
	var (
		out wshub.MessageItemPayload
		s   wshub.UserPayload
		r   wshub.UserPayload
	)
	out = wshub.MessageItemPayload{
		ID:         v.ID.String(),
		ProjectID:  v.ProjectID.String(),
		SenderID:   v.SenderID.String(),
		ReceiverID: v.ReceiverID.String(),
		Content:    v.Content,
		IsRead:     v.IsRead,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
	}
	if v.Sender != nil {
		s = userPayloadFrom(*v.Sender)
		out.Sender = &s
	}
	if v.Receiver != nil {
		r = userPayloadFrom(*v.Receiver)
		out.Receiver = &r
	}
	return out
}

func userPayloadFrom(v usecase.UserView) wshub.UserPayload {
	return wshub.UserPayload{
		ID:        v.ID.String(),
		Email:     v.Email,
		Name:      v.Name,
		Role:      v.Role,
		IsActive:  v.IsActive(),
		AvatarURL: v.AvatarURL,
	}
}
