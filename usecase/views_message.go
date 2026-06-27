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
}

type NotificationsWire struct {
	UnreadMessages int64 `json:"unread_messages"`
}

type UserWire struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
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

func UserWireFrom(v UserView) UserWire {
	return UserWire{
		ID:        v.ID.String(),
		Email:     v.Email,
		Name:      v.Name,
		Role:      v.Role,
		IsActive:  v.IsActive(),
		AvatarURL: v.AvatarURL,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func MessageItemWireFrom(v MessageItemView) MessageItemWire {
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

func MessageThreadWireFrom(t MessageThreadView) MessageThreadWire {
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

func ListMessageThreadsWireFrom(rows []MessageThreadView) ListMessageThreadsWire {
	var (
		out ListMessageThreadsWire
		i   int
	)
	out.Threads = make([]MessageThreadWire, len(rows))
	for i = range rows {
		out.Threads[i] = MessageThreadWireFrom(rows[i])
	}
	return out
}
