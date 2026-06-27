package wshub

import (
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
)

const EventSchemaVersion = 1

const ClientSchemaVersion = 1

var (
	PingInterval = 30 * time.Second
	ReadDeadline = 90 * time.Second
)

const (
	EventError          = "error"
	EventSessionReady   = "session_ready"
	EventMessageCreated = "message_created"
	EventReadUpdated    = "read_updated"
	EventTyping         = "typing"
	EventSubscribed     = "subscribed"
	EventUnsubscribed   = "unsubscribed"
	EventPong           = "pong"
)

const (
	ClientPing        = "ping"
	ClientTyping      = "typing"
	ClientSubscribe   = "subscribe"
	ClientUnsubscribe = "unsubscribe"
)

const (
	CodeInvalidJSON           = "INVALID_JSON"
	CodeInvalidProjectID      = "INVALID_PROJECT_ID"
	CodeForbiddenNotMember    = "FORBIDDEN_NOT_MEMBER"
	CodeUnknownType           = "UNKNOWN_TYPE"
	CodeInternal              = "INTERNAL"
	CodeSchemaVersion         = "SCHEMA_VERSION_MISMATCH"
	CodeMissingType           = "MISSING_TYPE"
	CodeMissingProjectID      = "MISSING_PROJECT_ID"
)

type ClientMessage struct {
	Type          string `json:"type"`
	ProjectID     string `json:"project_id"`
	IsTyping      *bool  `json:"is_typing,omitempty"`
	SchemaVersion int    `json:"schema_version,omitempty"`
}

type ErrorOutbound struct {
	Type          string  `json:"type"`
	SchemaVersion int     `json:"schema_version"`
	Code          string  `json:"code"`
	Message       string  `json:"message"`
	ProjectID     *string `json:"project_id,omitempty"`
}

type SessionReadyOutbound struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	UserID        uuid.UUID `json:"user_id"`
}

type ProjectAckOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	ProjectID     string `json:"project_id"`
}

type PongOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
}

type TypingOutbound struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	ProjectID     uuid.UUID `json:"project_id"`
	UserID        uuid.UUID `json:"user_id"`
	IsTyping      bool      `json:"is_typing"`
}

type MessageCreatedBroadcast struct {
	Type          string                  `json:"type"`
	SchemaVersion int                     `json:"schema_version"`
	ProjectID     uuid.UUID               `json:"project_id"`
	Message       usecase.MessageItemWire `json:"message"`
}

type ReadUpdatedBroadcast struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	ProjectID     uuid.UUID `json:"project_id"`
	MessageID     uuid.UUID `json:"message_id"`
	ReaderID      uuid.UUID `json:"reader_id"`
}
