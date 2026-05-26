-- +goose Up
CREATE TABLE chat_sessions (
    session_id       TEXT PRIMARY KEY,
    user_id          TEXT,
    auth_user_id     UUID,
    platform         TEXT DEFAULT 'web',
    is_human_mode    BOOLEAN DEFAULT false,
    last_inbound_at  TIMESTAMPTZ,
    last_ai_at       TIMESTAMPTZ,
    lock_until       TIMESTAMPTZ,
    lock_token       UUID,
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE chat_messages (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_id   TEXT REFERENCES chat_sessions(session_id) ON DELETE CASCADE,
    role         VARCHAR NOT NULL,
    content      TEXT NOT NULL,
    sender_type  VARCHAR,
    message_type TEXT NOT NULL DEFAULT 'text',
    file_path    TEXT,
    meta_data    JSONB DEFAULT '{}',
    media        JSONB,
    auth_user_id UUID,
    created_at   TIMESTAMPTZ DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
