CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    receiver_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_project_created ON messages (project_id, created_at DESC);
CREATE INDEX idx_messages_receiver_unread ON messages (receiver_id, is_read)
    WHERE is_read = FALSE;
CREATE INDEX idx_messages_project_participants ON messages (project_id, sender_id, receiver_id);

CREATE TABLE ws_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash VARCHAR NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ws_tokens_user_id ON ws_tokens (user_id);
CREATE INDEX idx_ws_tokens_expires_at ON ws_tokens (expires_at);
