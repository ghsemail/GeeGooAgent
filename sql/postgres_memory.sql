-- Semantic memory chunks (requires pgvector extension).
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS agent_memory_chunks (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL DEFAULT '',
    user_id     TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT 'session_summary',
    content     TEXT NOT NULL,
    embedding   vector(1536),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_chunks_session ON agent_memory_chunks (session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_chunks_user ON agent_memory_chunks (user_id, created_at DESC);

-- IVFFlat index optional after data exists:
-- CREATE INDEX idx_memory_chunks_embedding ON agent_memory_chunks USING ivfflat (embedding vector_cosine_ops);
