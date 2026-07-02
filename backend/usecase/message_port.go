package usecase

import (
	"context"

	"github.com/google/uuid"
)

type IMessageUsecase interface {
	ListThreads(ctx context.Context, userID uuid.UUID, q ListThreadsQuery) (ListThreadsResult, error)
	Send(ctx context.Context, senderID uuid.UUID, in SendMessageIn) (MessageItemView, error)
	MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta RequestMeta) error
	GetNotifications(ctx context.Context, userID uuid.UUID) (NotificationsView, error)
	CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error
	ConfirmWSSubscribe(ctx context.Context, userID, projectID uuid.UUID, meta RequestMeta) error
}

type IWSTokenUsecase interface {
	Issue(ctx context.Context, userID uuid.UUID) (string, error)
	Consume(ctx context.Context, plain string, meta RequestMeta) (uuid.UUID, error)
}
