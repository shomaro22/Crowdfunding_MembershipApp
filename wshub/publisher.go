package wshub

import (
	"context"
	"encoding/json"
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

type ProjectAccess interface {
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
}

type MessagePublisher struct {
	Hub    *ConnectionManager
	Access ProjectAccess
}

func NewMessagePublisher(hub *ConnectionManager, access ProjectAccess) *MessagePublisher {
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
	payload, err = json.Marshal(MessageCreatedBroadcast{
		Type:          EventMessageCreated,
		SchemaVersion: EventSchemaVersion,
		ProjectID:     view.ProjectID,
		Message:       usecase.MessageItemWireFrom(view),
	})
	if err != nil {
		return
	}
	p.Hub.BroadcastToProjectFiltered(view.ProjectID, payload, p.accessFilter(ctx, view.ProjectID))
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
	p.Hub.BroadcastToProjectFiltered(projectID, payload, p.accessFilter(ctx, projectID))
}

func (p *MessagePublisher) accessFilter(ctx context.Context, projectID uuid.UUID) func(uuid.UUID) bool {
	return func(userID uuid.UUID) bool {
		var (
			checkCtx context.Context
			cancel   context.CancelFunc
			err      error
		)
		checkCtx, cancel = context.WithTimeout(ctx, 3*time.Second)
		err = p.Access.CanSubscribeProject(checkCtx, projectID, userID)
		cancel()
		return err == nil
	}
}
