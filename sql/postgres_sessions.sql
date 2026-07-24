-- Chat session SSOT for PostgreSQL (mirrors SQLite chat_sessions).
CREATE TABLE IF NOT EXISTS chat_sessions (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL DEFAULT '',
    title             TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    step_counter      INTEGER NOT NULL DEFAULT 0,
    tags_json         JSONB NOT NULL DEFAULT '[]'::jsonb,
    summary           TEXT NOT NULL DEFAULT '',
    tool_names_json   JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    messages_json     JSONB NOT NULL DEFAULT '[]'::jsonb,
    step_records_json JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_updated ON chat_sessions (updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions (status);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions (user_id, updated_at DESC);

-- Full-text search (built-in; no pgvector required)
CREATE INDEX IF NOT EXISTS idx_chat_sessions_summary_trgm ON chat_sessions USING gin (summary gin_trgm_ops);
-- pg_trgm may be missing on minimal installs; ignore failure in app if needed
