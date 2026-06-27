package usecase

import (
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

type MessageItemView struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	SenderID   uuid.UUID
	ReceiverID uuid.UUID
	Content    string
	IsRead     bool
	ReadAt     *time.Time
	CreatedAt  time.Time
	Sender     *UserView
	Receiver   *UserView
}

type MessageThreadView struct {
	ProjectID    uuid.UUID
	ProjectTitle string
	Partner      UserView
	Messages     []MessageItemView
	UnreadCount  int
}

type NotificationsView struct {
	UnreadMessages int64
}

type ListThreadsQuery struct {
	Page              int
	PerPage           int
	MessagesPerThread int
}

type ListThreadsResult struct {
	Threads []MessageThreadView
	Total   int64
	Page    int
	PerPage int
}

func MessageItemFrom(msg *entity.Message) MessageItemView {
	if msg == nil {
		return MessageItemView{}
	}
	return MessageItemView{
		ID:         msg.ID,
		ProjectID:  msg.ProjectID,
		SenderID:   msg.SenderID,
		ReceiverID: msg.ReceiverID,
		Content:    msg.Content,
		IsRead:     msg.IsRead,
		ReadAt:     msg.ReadAt,
		CreatedAt:  msg.CreatedAt,
	}
}

func MessageItemWithUsers(msg *entity.Message, sender, receiver *UserView) MessageItemView {
	var out MessageItemView
	out = MessageItemFrom(msg)
	out.Sender = sender
	out.Receiver = receiver
	return out
}
