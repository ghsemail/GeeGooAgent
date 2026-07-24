-- GeeGoo Agent platform tables (PostgreSQL + optional pgvector).
-- Apply on GeeGooAgent server when migrating off SQLite-only cockpit metrics.
-- Session SSOT may remain SQLite initially; this schema supports multi-user ops UI.

CREATE TABLE IF NOT EXISTS agent_sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT '',
    title       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    source      TEXT NOT NULL DEFAULT 'dashboard',
    message_count INT NOT NULL DEFAULT 0,
    step_count    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_user_updated
    ON agent_sessions (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS agent_runs (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL DEFAULT '',
    topic       TEXT NOT NULL DEFAULT '',
    step_count  INT NOT NULL DEFAULT 0,
    failed      BOOLEAN NOT NULL DEFAULT FALSE,
    plan_pending BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_session
    ON agent_runs (session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_approvals (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,
    decision    TEXT NOT NULL,
    detail      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Optional: enable pgvector and semantic chunks (Phase 3)
-- CREATE EXTENSION IF NOT EXISTS vector;
-- CREATE TABLE IF NOT EXISTS agent_memory_chunks (
--     id          BIGSERIAL PRIMARY KEY,
--     user_id     TEXT NOT NULL DEFAULT '',
--     source      TEXT NOT NULL,
--     content     TEXT NOT NULL,
--     embedding   vector(1536),
--     created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- );
