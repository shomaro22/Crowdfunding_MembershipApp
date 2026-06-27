package controller

import (
	"time"

	"github.com/google/uuid"
)

const WSEventSchemaVersion = 1

const WSClientSchemaVersion = 1

var (
	WSPingInterval = 30 * time.Second
	WSReadDeadline = 90 * time.Second
)

const (
	WSEventError          = "error"
	WSEventSessionReady   = "session_ready"
	WSEventMessageCreated = "message_created"
	WSEventReadUpdated    = "read_updated"
	WSEventTyping         = "typing"
	WSEventSubscribed     = "subscribed"
	WSEventUnsubscribed   = "unsubscribed"
	WSEventPong           = "pong"
)

const (
	WSClientPing             = "ping"
	WSClientTyping           = "typing"
	WSClientSubscribe        = "subscribe"
	WSClientUnsubscribe      = "unsubscribe"
)

const (
	WSCodeInvalidJSON           = "INVALID_JSON"
	WSCodeInvalidProjectID      = "INVALID_PROJECT_ID"
	WSCodeForbiddenNotMember    = "FORBIDDEN_NOT_MEMBER"
	WSCodeUnknownType           = "UNKNOWN_TYPE"
	WSCodeInternal              = "INTERNAL"
	WSCodeSchemaVersion         = "SCHEMA_VERSION_MISMATCH"
	WSCodeMissingType           = "MISSING_TYPE"
	WSCodeMissingProjectID      = "MISSING_PROJECT_ID"
)

type wsClientMessage struct {
	Type          string `json:"type"`
	ProjectID     string `json:"project_id"`
	IsTyping      *bool  `json:"is_typing,omitempty"`
	SchemaVersion int    `json:"schema_version,omitempty"`
}

type wsErrorOutbound struct {
	Type          string  `json:"type"`
	SchemaVersion int     `json:"schema_version"`
	Code          string  `json:"code"`
	Message       string  `json:"message"`
	ProjectID     *string `json:"project_id,omitempty"`
}

type wsSessionReadyOutbound struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	UserID        uuid.UUID `json:"user_id"`
}

type wsProjectAckOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	ProjectID     string `json:"project_id"`
}

type wsPongOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
}

type typingOutboundPayload struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	ProjectID     uuid.UUID `json:"project_id"`
	UserID        uuid.UUID `json:"user_id"`
	IsTyping      bool      `json:"is_typing"`
}

type messageCreatedWSBroadcast struct {
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schema_version"`
	ProjectID     uuid.UUID       `json:"project_id"`
	Message       messageItemJSON `json:"message"`
}

type readUpdatedWSBroadcast struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	ProjectID     uuid.UUID `json:"project_id"`
	MessageID     uuid.UUID `json:"message_id"`
	ReaderID      uuid.UUID `json:"reader_id"`
}
