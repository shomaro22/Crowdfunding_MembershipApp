package controller

import (
	"time"

	"crowdfund-app/usecase"
)

type messageItemJSON struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  string    `json:"created_at"`
	Sender     *userJSON `json:"sender,omitempty"`
	Receiver   *userJSON `json:"receiver,omitempty"`
}

type messageThreadJSON struct {
	ProjectID    string            `json:"project_id"`
	ProjectTitle string            `json:"project_title"`
	Partner      userJSON          `json:"partner"`
	Messages     []messageItemJSON `json:"messages"`
	UnreadCount  int               `json:"unread_count"`
}

type listMessageThreadsResponse struct {
	Threads []messageThreadJSON `json:"threads"`
}

type notificationsResponse struct {
	UnreadMessages int64 `json:"unread_messages"`
}

func messageItemToJSON(v usecase.MessageItemView) messageItemJSON {
	var (
		out messageItemJSON
		s   userJSON
		r   userJSON
	)
	out = messageItemJSON{
		ID:         v.ID.String(),
		ProjectID:  v.ProjectID.String(),
		SenderID:   v.SenderID.String(),
		ReceiverID: v.ReceiverID.String(),
		Content:    v.Content,
		IsRead:     v.IsRead,
		CreatedAt:  v.CreatedAt.UTC().Format(time.RFC3339),
	}
	if v.Sender != nil {
		s = userToJSON(*v.Sender)
		out.Sender = &s
	}
	if v.Receiver != nil {
		r = userToJSON(*v.Receiver)
		out.Receiver = &r
	}
	return out
}

func messageThreadToJSON(t usecase.MessageThreadView) messageThreadJSON {
	var (
		out messageThreadJSON
		i   int
	)
	out = messageThreadJSON{
		ProjectID:    t.ProjectID.String(),
		ProjectTitle: t.ProjectTitle,
		Partner:      userToJSON(t.Partner),
		UnreadCount:  t.UnreadCount,
		Messages:     make([]messageItemJSON, len(t.Messages)),
	}
	for i = range t.Messages {
		out.Messages[i] = messageItemToJSON(t.Messages[i])
	}
	return out
}

func listMessageThreadsJSON(rows []usecase.MessageThreadView) listMessageThreadsResponse {
	var (
		out listMessageThreadsResponse
		i   int
	)
	out.Threads = make([]messageThreadJSON, len(rows))
	for i = range rows {
		out.Threads[i] = messageThreadToJSON(rows[i])
	}
	return out
}
