-- GeeGooAgent SQLite schema. Idempotent DDL consumed by internal/infra/db.go.

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_sessions (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    step_counter  INTEGER NOT NULL DEFAULT 0,
    tags_json     TEXT NOT NULL DEFAULT '[]',
    summary       TEXT NOT NULL DEFAULT '',
    tool_names_json TEXT NOT NULL DEFAULT '[]',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    messages_json TEXT NOT NULL DEFAULT '[]',
    step_records_json TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_updated ON chat_sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status   ON chat_sessions(status);

CREATE VIRTUAL TABLE IF NOT EXISTS chat_sessions_fts USING fts5(
    session_id UNINDEXED, title, summary
);

CREATE TABLE IF NOT EXISTS session_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    step        INTEGER NOT NULL,
    kind        TEXT NOT NULL,
    tool_name   TEXT NOT NULL DEFAULT '',
    tool_status TEXT NOT NULL DEFAULT '',
    summary     TEXT NOT NULL DEFAULT '',
    ts          TEXT NOT NULL,
    FOREIGN KEY(session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_events_session ON session_events(session_id, step);

CREATE TABLE IF NOT EXISTS evidence_records (
    id           TEXT PRIMARY KEY,
    run_id       TEXT NOT NULL,
    session_id   TEXT NOT NULL DEFAULT '',
    tool         TEXT NOT NULL,
    source       TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    summary      TEXT NOT NULL DEFAULT '',
    observed_at  TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_evidence_run     ON evidence_records(run_id);
CREATE INDEX IF NOT EXISTS idx_evidence_session ON evidence_records(session_id);
CREATE INDEX IF NOT EXISTS idx_evidence_source  ON evidence_records(source);

CREATE TABLE IF NOT EXISTS working_state (
    session_id   TEXT PRIMARY KEY,
    phase        TEXT NOT NULL DEFAULT 'init',
    working_json TEXT NOT NULL DEFAULT '{}',
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS checkpoints (
    session_id   TEXT PRIMARY KEY,
    step         INTEGER NOT NULL,
    skill        TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT '',
    last_tool    TEXT NOT NULL DEFAULT '',
    working_json TEXT NOT NULL DEFAULT '{}',
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS execution_events (
    rowid              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id         TEXT NOT NULL,
    step               INTEGER NOT NULL,
    step_name          TEXT NOT NULL DEFAULT '',
    tool               TEXT NOT NULL DEFAULT '',
    args_summary       TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT '',
    error              TEXT NOT NULL DEFAULT '',
    retry_count        INTEGER NOT NULL DEFAULT 0,
    started_at         TEXT NOT NULL,
    ended_at           TEXT NOT NULL DEFAULT '',
    duration_ms        INTEGER NOT NULL DEFAULT 0,
    checkpoint_id      TEXT NOT NULL DEFAULT '',
    supervisor_verdict TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_exec_session ON execution_events(session_id, step);

-- Eval test cases & run logs (SQLite parity with postgres_eval.sql).
CREATE TABLE IF NOT EXISTS agent_eval_cases (
    id                    TEXT PRIMARY KEY,
    user_id               TEXT NOT NULL DEFAULT '',
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL DEFAULT '',
    steps_json            TEXT NOT NULL DEFAULT '[]',
    supports_random_stock INTEGER NOT NULL DEFAULT 0,
    options_json          TEXT NOT NULL DEFAULT '{}',
    sort_order            INTEGER NOT NULL DEFAULT 0,
    enabled               INTEGER NOT NULL DEFAULT 1,
    created_at            TEXT NOT NULL,
    updated_at            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_cases_user_sort
    ON agent_eval_cases (user_id, sort_order, id);

CREATE TABLE IF NOT EXISTS agent_eval_runs (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL DEFAULT '',
    case_id       TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'idle',
    dual_model    INTEGER NOT NULL DEFAULT 0,
    model_slot_a  TEXT NOT NULL DEFAULT '',
    model_slot_b  TEXT NOT NULL DEFAULT '',
    duration_ms   INTEGER,
    error_text    TEXT NOT NULL DEFAULT '',
    logs_json     TEXT NOT NULL DEFAULT '[]',
    started_at    TEXT NOT NULL,
    ended_at      TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_runs_user_started
    ON agent_eval_runs (user_id, started_at DESC);

INSERT OR IGNORE INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at
) VALUES (
    'hello_then_stock_analysis',
    '',
    'Hello + 股票分析',
    '发送 hello 后分析指定或随机股票走势（默认单模型，在右侧 Dock Chat 展示完整过程）。',
    '["按配置清空 Chat 会话（仅 session，不影响评估日志）","发送 hello 与股票分析请求","等待回复完成并记录日志"]',
    1,
    '{"random_stock_enabled":true,"dual_model_eval":false,"session_cleanup":"before_run"}',
    0,
    1,
    datetime('now'),
    datetime('now')
);
