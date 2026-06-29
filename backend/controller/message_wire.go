package controller

import (
	"time"

	"crowdfund-app/usecase"
)

type sendMessageRequestJSON struct {
	ProjectID  string `json:"project_id"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
}

type MessageItemWire struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  string    `json:"created_at"`
	Sender     *UserWire `json:"sender,omitempty"`
	Receiver   *UserWire `json:"receiver,omitempty"`
}

type MessageThreadWire struct {
	ProjectID    string            `json:"project_id"`
	ProjectTitle string            `json:"project_title"`
	Partner      UserWire          `json:"partner"`
	Messages     []MessageItemWire `json:"messages"`
	UnreadCount  int               `json:"unread_count"`
}

type ListMessageThreadsWire struct {
	Threads []MessageThreadWire `json:"threads"`
	Total   int64               `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
}

type NotificationsWire struct {
	UnreadMessages int64 `json:"unread_messages"`
}

func MessageItemWireFrom(v usecase.MessageItemView) MessageItemWire {
	var (
		out MessageItemWire
		s   UserWire
		r   UserWire
	)
	out = MessageItemWire{
		ID:         v.ID.String(),
		ProjectID:  v.ProjectID.String(),
		SenderID:   v.SenderID.String(),
		ReceiverID: v.ReceiverID.String(),
		Content:    v.Content,
		IsRead:     v.IsRead,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
	}
	if v.Sender != nil {
		s = UserWireFrom(*v.Sender)
		out.Sender = &s
	}
	if v.Receiver != nil {
		r = UserWireFrom(*v.Receiver)
		out.Receiver = &r
	}
	return out
}

func MessageThreadWireFrom(t usecase.MessageThreadView) MessageThreadWire {
	var (
		out MessageThreadWire
		i   int
	)
	out = MessageThreadWire{
		ProjectID:    t.ProjectID.String(),
		ProjectTitle: t.ProjectTitle,
		Partner:      UserWireFrom(t.Partner),
		UnreadCount:  t.UnreadCount,
		Messages:     make([]MessageItemWire, len(t.Messages)),
	}
	for i = range t.Messages {
		out.Messages[i] = MessageItemWireFrom(t.Messages[i])
	}
	return out
}

func ListMessageThreadsWireFrom(result usecase.ListThreadsResult) ListMessageThreadsWire {
	var (
		out ListMessageThreadsWire
		i   int
	)
	out.Total = result.Total
	out.Page = result.Page
	out.PerPage = result.PerPage
	out.Threads = make([]MessageThreadWire, len(result.Threads))
	for i = range result.Threads {
		out.Threads[i] = MessageThreadWireFrom(result.Threads[i])
	}
	return out
}
