package wshub

import "github.com/google/uuid"

type UserPayload struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name,omitempty"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type MessageItemPayload struct {
	ID         string       `json:"id"`
	ProjectID  string       `json:"project_id"`
	SenderID   string       `json:"sender_id"`
	ReceiverID string       `json:"receiver_id"`
	Content    string       `json:"content"`
	IsRead     bool         `json:"is_read"`
	CreatedAt  string       `json:"created_at"`
	Sender     *UserPayload `json:"sender,omitempty"`
	Receiver   *UserPayload `json:"receiver,omitempty"`
}

type MessageCreatedOutbound struct {
	Type          string             `json:"type"`
	SchemaVersion int                `json:"schema_version"`
	ProjectID     uuid.UUID          `json:"project_id"`
	Message       MessageItemPayload `json:"message"`
}
