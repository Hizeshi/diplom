-- +goose Up

-- Store a SHA-256 hash of the anonymous session ownership token.
-- Anonymous clients receive the raw token on first message and must present it
-- on subsequent requests. We never store the raw token.
ALTER TABLE chat_sessions ADD COLUMN IF NOT EXISTS session_token_hash TEXT;

-- Indexes to speed up ownership lookups and admin/history queries.
CREATE INDEX IF NOT EXISTS chat_sessions_auth_user_id_idx ON chat_sessions (auth_user_id) WHERE auth_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS chat_sessions_updated_at_idx  ON chat_sessions (updated_at DESC);
CREATE INDEX IF NOT EXISTS chat_messages_session_created_idx ON chat_messages (session_id, created_at);

-- Role and message_type constraints.
ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_role_check
    CHECK (role IN ('user', 'assistant', 'system'));

ALTER TABLE chat_messages ADD CONSTRAINT chat_messages_type_check
    CHECK (message_type IN ('text', 'voice', 'photo', 'document'));

-- +goose Down
ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chat_messages_type_check;
ALTER TABLE chat_messages DROP CONSTRAINT IF EXISTS chat_messages_role_check;
DROP INDEX IF EXISTS chat_messages_session_created_idx;
DROP INDEX IF EXISTS chat_sessions_updated_at_idx;
DROP INDEX IF EXISTS chat_sessions_auth_user_id_idx;
ALTER TABLE chat_sessions DROP COLUMN IF EXISTS session_token_hash;
