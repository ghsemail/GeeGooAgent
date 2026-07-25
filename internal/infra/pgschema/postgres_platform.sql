-- GeeGoo Agent platform tables (PostgreSQL).
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
    session_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL DEFAULT '',
    topic       TEXT NOT NULL DEFAULT '',
    step_count  INT NOT NULL DEFAULT 0,
    failed      BOOLEAN NOT NULL DEFAULT FALSE,
    plan_pending BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_session
    ON agent_runs (session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_episodes (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL DEFAULT '',
    user_id     TEXT NOT NULL DEFAULT '',
    happened_at DATE NOT NULL DEFAULT CURRENT_DATE,
    summary     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_episodes_user_date
    ON agent_episodes (user_id, happened_at DESC);

-- Semantic memory: durable facts (Waku facts table parity).
CREATE TABLE IF NOT EXISTS agent_facts (
    id          BIGSERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT '',
    subject     TEXT NOT NULL,
    content     TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'user',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(subject, '') || ' ' || coalesce(content, ''))
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_agent_facts_user ON agent_facts (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_agent_facts_fts ON agent_facts USING GIN (search_vector);

CREATE TABLE IF NOT EXISTS agent_approvals (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,
    decision    TEXT NOT NULL,
    detail      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
