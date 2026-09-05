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
    '{"category":"general","random_stock_enabled":true,"dual_model_eval":false,"session_cleanup":"before_run"}',
    0,
    1,
    datetime('now'),
    datetime('now')
);

INSERT OR IGNORE INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at
) VALUES
(
    'strategy_signal_single', '', '单股 · 单策略 · 信号测试',
    '随机抽取一只股票与一项高级策略，调用 probe 并汇报买卖信号。',
    '["随机选股与策略","发送信号测试请求（含标的与策略名）","校验回复含信号/买卖信息"]',
    1,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"single","stock_count":1,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":80,"pass_keywords":["信号","买","卖"],"session_cleanup":"before_run"}',
    10, 1, datetime('now'), datetime('now')
),
(
    'strategy_signal_multi_strategy', '', '单股 · 多策略 · 信号测试',
    '同一只股票上对比 2 项随机策略的信号触发情况。',
    '["随机选股与 2 项策略","发送多策略对比 probe 请求","校验回复含对比/信号摘要"]',
    1,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"multi_strategy","stock_count":1,"strategy_count":2,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["信号","对比"],"session_cleanup":"before_run"}',
    11, 1, datetime('now'), datetime('now')
),
(
    'strategy_signal_multi_stock', '', '多股 · 单策略 · 信号测试',
    '同一策略在 2 只随机股票上分别做信号测试并对比。',
    '["随机选 2 只股票与 1 项策略","发送多标的信号测试请求","校验回复含各股信号对比"]',
    1,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"multi_stock","stock_count":2,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["信号","对比"],"session_cleanup":"before_run"}',
    12, 1, datetime('now'), datetime('now')
),
(
    'strategy_backtest_single', '', '单股 · 单策略 · 回测',
    '随机单股单策略跑 run_strategy_backtest，汇报收益与 log_id。',
    '["随机选股与策略","发送回测请求","校验回复含收益率与 log_id"]',
    1,
    '{"category":"strategy_backtest","task":"backtest","scenario":"single","stock_count":1,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":80,"pass_keywords":["回测","收益","log"],"session_cleanup":"before_run"}',
    20, 1, datetime('now'), datetime('now')
),
(
    'strategy_backtest_multi_strategy', '', '单股 · 多策略 · 回测',
    '同一只股票上对比 2 项随机策略的回测收益。',
    '["随机选股与 2 项策略","发送多策略回测对比请求","校验回复含收益对比与 log_id"]',
    1,
    '{"category":"strategy_backtest","task":"backtest","scenario":"multi_strategy","stock_count":1,"strategy_count":2,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["回测","对比","log"],"session_cleanup":"before_run"}',
    21, 1, datetime('now'), datetime('now')
),
(
    'strategy_backtest_multi_config', '', '单股 · 单策略 · 多止盈止损回测',
    '同一策略用两套止盈止损参数（如 5%/3% vs 7%/5%）跑回测并对比。',
    '["随机选股与策略","发送多止盈止损参数回测请求","校验回复含各套配置收益对比"]',
    1,
    '{"category":"strategy_backtest","task":"backtest","scenario":"multi_config","stock_count":1,"strategy_count":1,"config_variants":["止盈5%止损3%","止盈7%止损5%"],"random_stock_enabled":true,"min_reply_chars":120,"pass_keywords":["回测","对比","止盈"],"session_cleanup":"before_run"}',
    22, 1, datetime('now'), datetime('now')
);

DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_price_lookup', '', 'TurnPlan · 查腾讯股价', '真实 Chat 执行：发送「帮我查一下腾讯的股价」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我查一下腾讯的股价","expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":true,"forbid_tools":["run_strategy_backtest"],"require_tools":["search_code","get_mcp_analysis"],"min_reply_chars":20,"turn_id":"stock_price_lookup"}', 50, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_followup_technical', '', 'TurnPlan · 分析技术面跟进', '真实 Chat 执行：发送「可以，分析下技术面的价格和K线图」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","先发送 setup 话术建立上下文，再发送目标用户消息","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"可以，分析下技术面的价格和K线图","setup_messages":["帮我查一下腾讯的股价"],"expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":true,"forbid_tools":["run_strategy_backtest"],"require_tools":["search_code","get_mcp_analysis"],"min_reply_chars":20,"turn_id":"stock_followup_technical"}', 51, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_colloquial', '', 'TurnPlan · 口语换股票', '真实 Chat 执行：发送「中际旭创这边呢」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"中际旭创这边呢","expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":true,"forbid_tools":["run_strategy_backtest"],"require_tools":["search_code"],"min_reply_chars":20,"turn_id":"stock_colloquial"}', 52, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_signal_probe', '', 'TurnPlan · 测买卖点', '真实 Chat 执行：发送「帮我看看中际旭创有没有买卖点」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我看看中际旭创有没有买卖点","expect_domain":"signal_probe","expect_mode":"execute","expect_sop":true,"require_tools":["probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"signal_probe"}', 53, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_signal_probe_combo', '', 'TurnPlan · SAR+MACD 测点', '真实 Chat 执行：发送「就这个SAR加MACD组合信号，我想先测买卖点」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"就这个SAR加MACD组合信号，我想先测买卖点","expect_domain":"signal_probe","expect_mode":"execute","expect_sop":true,"require_tools":["probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"signal_probe_combo"}', 54, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_explicit', '', 'TurnPlan · 显式回测', '真实 Chat 执行：发送「帮我用SAR加MACD回测一下小米」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我用SAR加MACD回测一下小米","expect_domain":"backtest_run","expect_mode":"execute","expect_sop":true,"require_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"backtest_explicit"}', 55, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_colloquial', '', 'TurnPlan · 口语回测', '真实 Chat 执行：发送「帮我回测一下中际旭创」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我回测一下中际旭创","expect_domain":"backtest_run","expect_mode":"execute","expect_sop":true,"require_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"backtest_colloquial"}', 56, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_ambiguous_bare_macd', '', 'TurnPlan · 模糊 MACD', '真实 Chat 执行：发送「这个MACD信号怎么弄比较好」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"这个MACD信号怎么弄比较好","expect_domain":"ambiguous","expect_mode":"clarify","expect_sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"ambiguous_bare_macd"}', 57, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_compound_analysis_backtest', '', 'TurnPlan · 分析+回测复合', '真实 Chat 执行：发送「帮我把中际旭创分析一下，再跑个回测看看」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我把中际旭创分析一下，再跑个回测看看","expect_domain":"ambiguous","expect_mode":"clarify","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"compound_analysis_backtest"}', 58, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_chat_definition', '', 'TurnPlan · MACD 释义', '真实 Chat 执行：发送「MACD 指标是什么意思」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"MACD 指标是什么意思","expect_domain":"chat","expect_mode":"talk","expect_sop":false,"forbid_tools":["run_strategy_backtest","get_mcp_analysis"],"min_reply_chars":20,"turn_id":"chat_definition"}', 59, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_chat_signal_quality', '', 'TurnPlan · 信号准吗', '真实 Chat 执行：发送「这个信号准吗」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"这个信号准吗","expect_domain":"chat","expect_mode":"talk","expect_sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"chat_signal_quality"}', 60, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_reminder_list', '', 'TurnPlan · Reminder 列表', '真实 Chat 执行：发送「我现在有哪些reminder」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"我现在有哪些reminder","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_dca_reminders"],"min_reply_chars":20,"turn_id":"bot_reminder_list"}', 61, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_grid_pnl', '', 'TurnPlan · 网格 Bot 盈亏', '真实 Chat 执行：发送「帮我查看腾讯网格Bot的盈亏」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我查看腾讯网格Bot的盈亏","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_grid_bots"],"min_reply_chars":20,"turn_id":"bot_grid_pnl"}', 62, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_smarttrade_list', '', 'TurnPlan · SmartTrade 列表', '真实 Chat 执行：发送「帮我查一下我有哪些SmartTrade」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我查一下我有哪些SmartTrade","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_smart_trades"],"min_reply_chars":20,"turn_id":"bot_smarttrade_list"}', 63, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_history', '', 'TurnPlan · 回测历史', '真实 Chat 执行：发送「上次回测结果怎么样」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"上次回测结果怎么样","expect_domain":"backtest_history","expect_mode":"gather","expect_sop":false,"require_tools":["list_strategy_backtest_logs"],"min_reply_chars":20,"turn_id":"backtest_history"}', 64, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_report_lookup', '', 'TurnPlan · 盘前报告', '真实 Chat 执行：发送「今天盘前写了什么」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"今天盘前写了什么","expect_domain":"report_lookup","expect_mode":"gather","expect_sop":false,"require_tools":["get_stock_premarket_reports"],"min_reply_chars":20,"turn_id":"report_lookup"}', 65, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_knowledge_lookup', '', 'TurnPlan · 知识库检索', '真实 Chat 执行：发送「按知识库讲 4H MACD」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"按知识库讲 4H MACD","expect_domain":"knowledge","expect_mode":"gather","expect_sop":false,"require_tools":["search_knowledge"],"min_reply_chars":20,"turn_id":"knowledge_lookup"}', 66, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_news_lookup', '', 'TurnPlan · 新闻查询', '真实 Chat 执行：发送「有什么新闻」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"有什么新闻","expect_domain":"news","expect_mode":"gather","expect_sop":false,"require_tools":["fetch_market_news"],"min_reply_chars":20,"turn_id":"news_lookup"}', 67, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_dca_grid_backtest', '', 'TurnPlan · DCA 回测', '真实 Chat 执行：发送「帮我做 dca 定投回测」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","按 session 话术发送用户消息（含必要的 setup 轮次）","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我做 dca 定投回测","expect_domain":"dca_grid","expect_mode":"execute","expect_sop":false,"require_tools":["generate_dca_strategy"],"min_reply_chars":20,"turn_id":"dca_grid_backtest"}', 68, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_sticky_symbol_switch', '', 'TurnPlan · 换贵州茅台', '真实 Chat 执行：发送「那就换成贵州茅台吧」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","先发送 setup 话术建立上下文，再发送目标用户消息","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"那就换成贵州茅台吧","setup_messages":["帮我分析一下中际旭创"],"expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":true,"require_tools":["search_code"],"min_reply_chars":20,"turn_id":"sticky_symbol_switch"}', 69, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_after_analysis', '', 'TurnPlan · 分析后回测', '真实 Chat 执行：发送「好，那帮我回测一下小米」，校验 turn_plan 路由与工具调用。', '["清空 Dock Chat 会话","先发送 setup 话术建立上下文，再发送目标用户消息","校验 turn_plan domain/mode/SOP 与实际工具调用"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"好，那帮我回测一下小米","setup_messages":["帮我分析一下小米"],"expect_domain":"backtest_run","expect_mode":"execute","expect_sop":true,"require_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"backtest_after_analysis"}', 70, 1, datetime('now'), datetime('now'));
