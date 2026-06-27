package controller

import (
	"context"
	"encoding/json"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type MessageWSPublisher struct {
	Hub ProjectBroadcastHub
}

func NewMessageWSPublisher(hub ProjectBroadcastHub) *MessageWSPublisher {
	return &MessageWSPublisher{Hub: hub}
}

func (p *MessageWSPublisher) OnMessageCreated(ctx context.Context, view usecase.MessageItemView) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil {
		return
	}
	_ = ctx
	payload, err = json.Marshal(messageCreatedWSBroadcast{
		Type:          WSEventMessageCreated,
		SchemaVersion: WSEventSchemaVersion,
		ProjectID:     view.ProjectID,
		Message:       messageItemToJSON(view),
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProject(view.ProjectID, payload)
}

func (p *MessageWSPublisher) OnMessageRead(ctx context.Context, projectID, messageID, readerID uuid.UUID) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil {
		return
	}
	_ = ctx
	payload, err = json.Marshal(readUpdatedWSBroadcast{
		Type:          WSEventReadUpdated,
		SchemaVersion: WSEventSchemaVersion,
		ProjectID:     projectID,
		MessageID:     messageID,
		ReaderID:      readerID,
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProject(projectID, payload)
}
