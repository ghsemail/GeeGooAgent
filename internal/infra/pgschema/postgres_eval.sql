-- Eval test cases & run logs (PostgreSQL). Idempotent DDL.

CREATE TABLE IF NOT EXISTS agent_eval_cases (
    id                    TEXT PRIMARY KEY,
    user_id               TEXT NOT NULL DEFAULT '',
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL DEFAULT '',
    steps_json            TEXT NOT NULL DEFAULT '[]',
    supports_random_stock BOOLEAN NOT NULL DEFAULT FALSE,
    options_json          TEXT NOT NULL DEFAULT '{}',
    sort_order            INT NOT NULL DEFAULT 0,
    enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_cases_user_sort
    ON agent_eval_cases (user_id, sort_order, id);

CREATE TABLE IF NOT EXISTS agent_eval_runs (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL DEFAULT '',
    case_id       TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'idle',
    dual_model    BOOLEAN NOT NULL DEFAULT FALSE,
    model_slot_a  TEXT NOT NULL DEFAULT '',
    model_slot_b  TEXT NOT NULL DEFAULT '',
    duration_ms   INT,
    error_text    TEXT NOT NULL DEFAULT '',
    logs_json     TEXT NOT NULL DEFAULT '[]',
    started_at    TIMESTAMPTZ NOT NULL,
    ended_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_runs_user_started
    ON agent_eval_runs (user_id, started_at DESC);

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order
) VALUES (
    'hello_then_stock_analysis',
    '',
    'Hello + 股票分析',
    '发送 hello 后分析指定或随机股票走势（默认单模型，在右侧 Dock Chat 展示完整过程）。',
    '["按配置清空 Chat 会话（仅 session，不影响评估日志）","发送 hello 与股票分析请求","等待回复完成并记录日志"]',
    TRUE,
    '{"random_stock_enabled":true,"dual_model_eval":false,"session_cleanup":"before_run"}',
    0
) ON CONFLICT (id) DO NOTHING;
