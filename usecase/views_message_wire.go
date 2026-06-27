package usecase

import (
	"time"
)

type MessageItemWire struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	SenderID   string     `json:"sender_id"`
	ReceiverID string     `json:"receiver_id"`
	Content    string     `json:"content"`
	IsRead     bool       `json:"is_read"`
	CreatedAt  string     `json:"created_at"`
	Sender     *UserWire  `json:"sender,omitempty"`
	Receiver   *UserWire  `json:"receiver,omitempty"`
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
