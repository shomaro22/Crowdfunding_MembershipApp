package wshub

import (
	"encoding/json"

	"github.com/google/uuid"
)

type MessagePublisher struct {
	Hub *ConnectionManager
}

func NewMessagePublisher(hub *ConnectionManager) *MessagePublisher {
	return &MessagePublisher{Hub: hub}
}

func (p *MessagePublisher) PublishMessageCreated(projectID uuid.UUID, item MessageItemPayload) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil {
		return
	}
	payload, err = json.Marshal(MessageCreatedOutbound{
		Type:          EventMessageCreated,
		SchemaVersion: EventSchemaVersion,
		ProjectID:     projectID,
		Message:       item,
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProjectAsync(projectID, payload)
}

func (p *MessagePublisher) PublishReadUpdated(projectID, messageID, readerID uuid.UUID) {
	var (
		payload []byte
		err     error
	)
	if p == nil || p.Hub == nil {
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
	p.Hub.BroadcastToProjectAsync(projectID, payload)
}
