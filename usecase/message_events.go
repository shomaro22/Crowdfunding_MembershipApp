package usecase

import (
	"context"

	"github.com/google/uuid"
)

type MessageEventHook interface {
	OnMessageCreated(ctx context.Context, view MessageItemView)
	OnMessageRead(ctx context.Context, projectID, messageID, readerID uuid.UUID)
}
