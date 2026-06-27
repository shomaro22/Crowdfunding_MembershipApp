package wshub

import (
	"context"
	"encoding/json"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type MessageSubscriberFilter interface {
	SubscriberFilter(ctx context.Context, projectID uuid.UUID) func(uuid.UUID) bool
}

type MessagePublisher struct {
	Hub    *ConnectionManager
	Access MessageSubscriberFilter
}

func NewMessagePublisher(hub *ConnectionManager, access MessageSubscriberFilter) *MessagePublisher {
	return &MessagePublisher{Hub: hub, Access: access}
}

func (p *MessagePublisher) OnMessageCreated(ctx context.Context, view usecase.MessageItemView) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil || p.Access == nil {
		return
	}
	payload, err = json.Marshal(struct {
		Type          string                  `json:"type"`
		SchemaVersion int                     `json:"schema_version"`
		ProjectID     uuid.UUID               `json:"project_id"`
		Message       usecase.MessageItemWire `json:"message"`
	}{
		Type:          EventMessageCreated,
		SchemaVersion: EventSchemaVersion,
		ProjectID:     view.ProjectID,
		Message:       usecase.MessageItemWireFrom(view),
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProjectFiltered(view.ProjectID, payload, p.Access.SubscriberFilter(ctx, view.ProjectID))
}

func (p *MessagePublisher) OnMessageRead(ctx context.Context, projectID, messageID, readerID uuid.UUID) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil || p.Access == nil {
		return
	}
	payload, err = json.Marshal(ReadUpdatedBroadcast{
		Type:          EventReadUpdated,
		SchemaVersion: EventSchemaVersion,
		ProjectID:     projectID,
		MessageID:     messageID,
		ReaderID:      readerID,
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProjectFiltered(projectID, payload, p.Access.SubscriberFilter(ctx, projectID))
}
