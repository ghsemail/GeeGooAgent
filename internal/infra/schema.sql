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
    dialogue_snapshot_json TEXT NOT NULL DEFAULT '[]',
    summary_json  TEXT NOT NULL DEFAULT '{}',
    started_at    TEXT NOT NULL,
    ended_at      TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS agent_eval_run_checks (
    id              TEXT PRIMARY KEY,
    run_id          TEXT NOT NULL,
    case_id         TEXT NOT NULL DEFAULT '',
    session_id      TEXT NOT NULL DEFAULT '',
    check_type      TEXT NOT NULL,
    passed          INTEGER NOT NULL,
    score           REAL,
    expected_json   TEXT NOT NULL DEFAULT '{}',
    actual_json     TEXT NOT NULL DEFAULT '{}',
    detail          TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_run_checks_run
    ON agent_eval_run_checks (run_id, check_type);

CREATE INDEX IF NOT EXISTS idx_agent_eval_runs_user_started
    ON agent_eval_runs (user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_eval_suites (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL DEFAULT '',
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    category_ids_json  TEXT NOT NULL DEFAULT '[]',
    case_ids_json      TEXT NOT NULL DEFAULT '[]',
    created_at         TEXT NOT NULL,
    updated_at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_suites_user
    ON agent_eval_suites (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS agent_eval_jobs (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL DEFAULT '',
    suite_id           TEXT NOT NULL DEFAULT '',
    title              TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'queued',
    category_ids_json  TEXT NOT NULL DEFAULT '[]',
    case_ids_json      TEXT NOT NULL DEFAULT '[]',
    total              INTEGER NOT NULL DEFAULT 0,
    passed             INTEGER NOT NULL DEFAULT 0,
    failed             INTEGER NOT NULL DEFAULT 0,
    error_text         TEXT NOT NULL DEFAULT '',
    started_at         TEXT NOT NULL DEFAULT '',
    ended_at           TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_jobs_user
    ON agent_eval_jobs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_eval_job_items (
    id            TEXT PRIMARY KEY,
    job_id        TEXT NOT NULL,
    case_id       TEXT NOT NULL,
    title         TEXT NOT NULL DEFAULT '',
    category      TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'pending',
    session_id    TEXT NOT NULL DEFAULT '',
    run_id        TEXT NOT NULL DEFAULT '',
    detail        TEXT NOT NULL DEFAULT '',
    summary_json  TEXT NOT NULL DEFAULT '{}',
    checks_json   TEXT NOT NULL DEFAULT '[]',
    loop_error    TEXT NOT NULL DEFAULT '',
    duration_ms   INTEGER,
    started_at    TEXT NOT NULL DEFAULT '',
    ended_at      TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_eval_job_items_job
    ON agent_eval_job_items (job_id, case_id);

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

DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%';
DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing';
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_price', '', 'TurnPlan · 单轮 · 查股价', '独立 session：查询腾讯控股现价（snapshot，非 MCP 分析）。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我查一下腾讯控股现在的股价」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我查一下腾讯控股现在的股价","expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"execution_profile":"stock_analysis.price_snapshot","min_reply_chars":20,"turn_id":"stock_price","dialogue":[{"role":"user","text":"帮我查一下腾讯控股现在的股价","judge":true}],"expect_reply":{"rubric":"应给出腾讯控股的现价或行情信息（价格、涨跌幅等），语气自然。","must_cover":["腾讯"]},"expect_intent":{"domain":"stock_analysis","mode":"gather","sop":false,"act":"quote_price"},"expect_execution":{"profile":"stock_analysis.price_snapshot","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"stock_analysis","mode":"gather","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 50, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_price_trend', '', 'TurnPlan · 单轮 · 分析价格走势', '独立 session：分析腾讯最近一个月价格走势（MCP）。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我分析下腾讯最近一个月的价格走势」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我分析下腾讯最近一个月的价格走势","expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"execution_profile":"stock_analysis.technical_full","min_reply_chars":20,"turn_id":"stock_price_trend","dialogue":[{"role":"user","text":"帮我分析下腾讯最近一个月的价格走势","judge":true}],"expect_reply":{"rubric":"应分析腾讯最近一个月的价格走势（趋势、涨跌、关键价位等），走 MCP 深度分析而非只报现价。","must_cover":["腾讯","走势"]},"expect_intent":{"domain":"stock_analysis","mode":"gather","sop":false,"act":"technical_analysis"},"expect_execution":{"profile":"stock_analysis.technical_full","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"stock_analysis","mode":"gather","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 51, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_technical_chain', '', 'TurnPlan · 多轮 · 查价后看K线', '同 session：先查腾讯股价，再自然续问 K 线图/技术面分析。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我查一下腾讯控股现在的股价」 → 「再帮我分析一下腾讯的K线图」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"再帮我分析一下腾讯的K线图","setup_messages":["帮我查一下腾讯控股现在的股价"],"expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"execution_profile":"stock_analysis.technical_full","min_reply_chars":20,"turn_id":"stock_technical_chain","dialogue":[{"role":"user","text":"帮我查一下腾讯控股现在的股价"},{"role":"user","text":"再帮我分析一下腾讯的K线图","judge":true}],"expect_reply":{"rubric":"在已查腾讯股价的基础上，应补充 K 线图或技术面分析，而非只重复报价。","must_cover":["腾讯"]},"expect_intent":{"domain":"stock_analysis","mode":"gather","sop":false,"act":"technical_analysis"},"expect_execution":{"profile":"stock_analysis.technical_full","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"stock_analysis","mode":"gather","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 52, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_symbol_switch', '', 'TurnPlan · 多轮 · 切换分析标的', '同 session：先问中际旭创价格趋势，再切换分析贵州茅台。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我分析一下中际旭创的价格趋势」 → 「再帮我分析一下贵州茅台」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"再帮我分析一下贵州茅台","setup_messages":["帮我分析一下中际旭创的价格趋势"],"expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"execution_profile":"stock_analysis.symbol_resolve","min_reply_chars":20,"turn_id":"stock_symbol_switch","dialogue":[{"role":"user","text":"帮我分析一下中际旭创的价格趋势"},{"role":"user","text":"再帮我分析一下贵州茅台","judge":true}],"expect_reply":{"rubric":"应识别用户切换到贵州茅台，并给出茅台相关分析或行情，而非继续只聊中际旭创。","must_cover":["茅台"]},"expect_intent":{"domain":"stock_analysis","mode":"gather","sop":false,"act":"symbol_resolve"},"expect_execution":{"profile":"stock_analysis.symbol_resolve","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"stock_analysis","mode":"gather","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 53, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_colloquial_ref', '', 'TurnPlan · 多轮 · 代词指代续问', '同 session：建立中际旭创价格走势上下文后，用「它」续问信号趋势。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我分析一下中际旭创的价格走势」 → 「它最近的信号趋势怎么样」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"它最近的信号趋势怎么样","setup_messages":["帮我分析一下中际旭创的价格走势"],"expect_domain":"stock_analysis","expect_mode":"gather","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"execution_profile":"stock_analysis.context_followup","min_reply_chars":20,"turn_id":"stock_colloquial_ref","dialogue":[{"role":"user","text":"帮我分析一下中际旭创的价格走势"},{"role":"user","text":"它最近的信号趋势怎么样","judge":true}],"expect_reply":{"rubric":"应理解「它」指代上一轮的中际旭创，并继续给出该标的信号趋势或技术面解读，而非换标的或跑回测。","must_cover":["中际","信号"]},"expect_intent":{"domain":"stock_analysis","mode":"gather","sop":false,"act":"context_followup"},"expect_execution":{"profile":"stock_analysis.context_followup","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"stock_analysis","mode":"gather","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 54, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_signal_catalog_list', '', 'TurnPlan · 单轮 · 列信号策略', '独立 session：列举可用信号/组合策略，plan 路由 dca_grid/gather 后由模型调 catalog 工具。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我看看我有哪些信号策略」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我看看我有哪些信号策略","expect_domain":"dca_grid","expect_mode":"gather","expect_sop":false,"require_tools":["get_signal_combinations"],"min_reply_chars":20,"turn_id":"signal_catalog_list","dialogue":[{"role":"user","text":"帮我看看我有哪些信号策略","judge":true}],"expect_reply":{"rubric":"应列出或摘要用户可用的信号/组合策略，语气自然，不应直接跑回测或 probe。","must_cover":["信号"],"must_not":["开始回测"]},"expect_intent":{"domain":"dca_grid","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["get_signal_combinations"]},"expect_routing":{"domain":"dca_grid","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 55, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_signal_list_then_probe', '', 'TurnPlan · 多轮 · 列策略后测买卖点', '同 session：先列出可用信号策略，再指定 SAR+MACD 测中际旭创买卖点。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我看看我有哪些信号策略」 → 「我想用SAR加MACD组合，测一下中际旭创有没有买卖点」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"我想用SAR加MACD组合，测一下中际旭创有没有买卖点","setup_messages":["帮我看看我有哪些信号策略"],"expect_domain":"signal_probe","expect_mode":"execute","expect_sop":false,"require_tools":["probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"signal_list_then_probe","dialogue":[{"role":"user","text":"帮我看看我有哪些信号策略"},{"role":"user","text":"我想用SAR加MACD组合，测一下中际旭创有没有买卖点","judge":true}],"expect_reply":{"rubric":"在用户已看过策略列表后，应确认 SAR+MACD 组合并对中际旭创做买卖点探测（含买/卖/暂无信号等），不应直接跑完整回测。","must_cover":["中际"],"must_not":["开始回测"]},"expect_intent":{"domain":"signal_probe","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["probe_bot_signal_series"]},"expect_routing":{"domain":"signal_probe","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 56, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_signal_probe_direct', '', 'TurnPlan · 单轮 · 直接测买卖点', '独立 session：显式指定标的与 signal_probe 意图；若 Agent clarify 策略则自动补默认选项。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我看看中际旭创有没有买卖点」","若 Agent clarify：自动回复「用SAR加MACD组合测买卖点」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我看看中际旭创有没有买卖点","expect_domain":"signal_probe","expect_mode":"execute","expect_sop":false,"require_tools":["probe_bot_signal_series"],"clarify_reply":"用SAR加MACD组合测买卖点","min_reply_chars":20,"turn_id":"signal_probe_direct","dialogue":[{"role":"user","text":"帮我看看中际旭创有没有买卖点"},{"role":"user","text":"用SAR加MACD组合测买卖点","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"应对中际旭创执行或汇报买卖点探测结果，包含信号方向或暂无信号说明。","must_cover":["中际"]},"expect_intent":{"domain":"signal_probe","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["probe_bot_signal_series"]},"expect_routing":{"domain":"signal_probe","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 57, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_explicit', '', 'TurnPlan · 单轮 · 显式回测', '独立 session：指定 SAR+MACD 回测小米；若 Agent clarify 周期/参数则自动补默认选项。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我用SAR加MACD回测一下小米」","若 Agent clarify：自动回复「用默认参数，最近3个月日线」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我用SAR加MACD回测一下小米","expect_domain":"backtest_run","expect_mode":"execute","expect_sop":false,"require_tools":["run_strategy_backtest"],"clarify_reply":"用默认参数，最近3个月日线","min_reply_chars":20,"turn_id":"backtest_explicit","dialogue":[{"role":"user","text":"帮我用SAR加MACD回测一下小米"},{"role":"user","text":"用默认参数，最近3个月日线","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"应确认 SAR+MACD 回测小米并已发起或汇报回测结果/进度，而非仅做股价查询。","must_cover":["小米","回测"]},"expect_intent":{"domain":"backtest_run","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"backtest_run","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 58, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_colloquial', '', 'TurnPlan · 单轮 · 口语回测', '独立 session：省略策略名但仍应路由到 backtest_run；若 Agent clarify 策略则自动补默认选项。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我回测一下中际旭创」","若 Agent clarify：自动回复「用SAR加MACD组合回测」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我回测一下中际旭创","expect_domain":"backtest_run","expect_mode":"execute","expect_sop":false,"require_tools":["run_strategy_backtest"],"clarify_reply":"用SAR加MACD组合回测","min_reply_chars":20,"turn_id":"backtest_colloquial","dialogue":[{"role":"user","text":"帮我回测一下中际旭创"},{"role":"user","text":"用SAR加MACD组合回测","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"应识别回测意图并针对中际旭创发起或说明回测，而非只做静态分析。","must_cover":["中际","回测"]},"expect_intent":{"domain":"backtest_run","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"backtest_run","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 59, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_analysis_then_backtest', '', 'TurnPlan · 多轮 · 分析后回测', '同 session：先分析小米，再在同一语境下发起回测。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我分析一下小米」 → 「接着用SAR加MACD帮小米跑个回测」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"接着用SAR加MACD帮小米跑个回测","setup_messages":["帮我分析一下小米"],"expect_domain":"backtest_run","expect_mode":"execute","expect_sop":false,"require_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"analysis_then_backtest","dialogue":[{"role":"user","text":"帮我分析一下小米"},{"role":"user","text":"接着用SAR加MACD帮小米跑个回测","judge":true}],"expect_reply":{"rubric":"在已分析小米的语境下，应发起 SAR+MACD 回测或明确回测执行与结果摘要。","must_cover":["回测"]},"expect_intent":{"domain":"backtest_run","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"backtest_run","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 60, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_strategy_list_then_backtest', '', 'TurnPlan · 多轮 · 选策略后回测', '同 session：先问可回测策略，再指定 SAR+MACD 回测中际旭创。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我看看有哪些可以回测的策略」 → 「用SAR加MACD组合回测中际旭创」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"用SAR加MACD组合回测中际旭创","setup_messages":["帮我看看有哪些可以回测的策略"],"expect_domain":"backtest_run","expect_mode":"execute","expect_sop":false,"require_tools":["run_strategy_backtest"],"min_reply_chars":20,"turn_id":"strategy_list_then_backtest","dialogue":[{"role":"user","text":"帮我看看有哪些可以回测的策略"},{"role":"user","text":"用SAR加MACD组合回测中际旭创","judge":true}],"expect_reply":{"rubric":"在用户已询问可回测策略后，应确认 SAR+MACD 并对中际旭创执行回测或汇报回测结果。","must_cover":["中际","回测"]},"expect_intent":{"domain":"backtest_run","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"backtest_run","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 61, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_ambiguous_bare_macd', '', 'TurnPlan · 两轮 · 模糊 MACD 信号', '独立 session：首轮应澄清信号库里的具体 MACD 组合；选定后按该信号策略说明讲解用法。', '["新 session：运行前清空 Dock Chat","首轮发送：「这个MACD信号平时该怎么用比较好」","若 Agent 展示澄清选项，用户选择：「SAR信号搭配MACD直方图趋势」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"这个MACD信号平时该怎么用比较好","expect_domain":"ambiguous","expect_mode":"clarify","expect_sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"],"clarify_reply":"SAR信号搭配MACD直方图趋势","min_reply_chars":20,"turn_id":"ambiguous_bare_macd","dialogue":[{"role":"user","text":"这个MACD信号平时该怎么用比较好"},{"role":"user","text":"SAR信号搭配MACD直方图趋势","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"用户已选定具体 MACD 组合信号（如 SAR+MACD）后，应基于该信号/策略说明讲解日常用法、适用场景或注意事项，而非直接测点或回测。","must_cover":["MACD","SAR"],"must_not":["回测已完成","开始回测"]},"expect_intent":{"domain":"ambiguous","mode":"clarify","sop":false},"expect_execution":{"profile":"","forbid_tools":["run_strategy_backtest","probe_bot_signal_series"]},"expect_routing":{"domain":"ambiguous","mode":"clarify","sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 62, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_compound_analysis_backtest', '', 'TurnPlan · 两轮 · 分析+回测复合', '独立 session：一句话同时含分析与回测时，首轮应澄清先做哪一步；用户选择「先只做分析」后应分析中际旭创。', '["新 session：运行前清空 Dock Chat","首轮发送：「帮我把中际旭创分析一下，然后再跑个回测看看效果」","若 Agent 展示澄清选项，用户选择：「先只做分析」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我把中际旭创分析一下，然后再跑个回测看看效果","expect_domain":"ambiguous","expect_mode":"clarify","expect_sop":false,"forbid_tools":["run_strategy_backtest"],"clarify_reply":"先只做分析","execution_profile":"stock_analysis.symbol_resolve","min_reply_chars":20,"turn_id":"compound_analysis_backtest","dialogue":[{"role":"user","text":"帮我把中际旭创分析一下，然后再跑个回测看看效果"},{"role":"user","text":"先只做分析","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"复合意图下用户选择「先只做分析」后，应针对中际旭创给出分析或行情解读，不应未经确认直接跑回测。","must_cover":["中际"],"must_not":["开始回测"]},"expect_intent":{"domain":"ambiguous","mode":"clarify","sop":false},"expect_execution":{"profile":"stock_analysis.symbol_resolve","forbid_tools":["run_strategy_backtest"]},"expect_routing":{"domain":"ambiguous","mode":"clarify","sop":false,"forbid_tools":["run_strategy_backtest"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 63, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_stock_quote_ambiguous', '', 'TurnPlan · 两轮 · 股价灰区', '独立 session：口语股价问法应澄清「现价 vs 走势分析」；用户选择「只要当前价」后应报腾讯现价。', '["新 session：运行前清空 Dock Chat","首轮发送：「腾讯股价怎么样」","若 Agent 展示澄清选项，用户选择：「只要当前价」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"腾讯股价怎么样","expect_domain":"ambiguous","expect_mode":"clarify","expect_sop":false,"forbid_tools":["run_strategy_backtest","get_mcp_analysis"],"clarify_reply":"只要当前价","execution_profile":"stock_analysis.price_snapshot","min_reply_chars":20,"turn_id":"stock_quote_ambiguous","dialogue":[{"role":"user","text":"腾讯股价怎么样"},{"role":"user","text":"只要当前价","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"用户已选择「只要当前价」后，应给出腾讯控股的现价或简要行情（价格、涨跌幅等），而非展开 MCP 深度分析或回测。","must_cover":["腾讯"],"must_not":["开始回测"]},"expect_intent":{"domain":"ambiguous","mode":"clarify","sop":false},"expect_execution":{"profile":"stock_analysis.price_snapshot","forbid_tools":["run_strategy_backtest","get_mcp_analysis"]},"expect_routing":{"domain":"ambiguous","mode":"clarify","sop":false,"forbid_tools":["run_strategy_backtest","get_mcp_analysis"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 64, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_chat_definition', '', 'TurnPlan · 单轮 · 指标释义', '独立 session：纯知识问答，plan 路由 chat/talk，不调用业务工具。', '["新 session：运行前清空 Dock Chat","单轮发送：「MACD 指标是什么意思」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"MACD 指标是什么意思","expect_domain":"chat","expect_mode":"talk","expect_sop":false,"forbid_tools":["run_strategy_backtest","get_mcp_analysis"],"min_reply_chars":20,"turn_id":"chat_definition","dialogue":[{"role":"user","text":"MACD 指标是什么意思","judge":true}],"expect_reply":{"rubric":"应用通俗语言解释 MACD 指标含义，不应调用行情或回测工具。","must_cover":["MACD"]},"expect_intent":{"domain":"chat","mode":"talk","sop":false},"expect_execution":{"profile":"","forbid_tools":["run_strategy_backtest","get_mcp_analysis"]},"expect_routing":{"domain":"chat","mode":"talk","sop":false,"forbid_tools":["run_strategy_backtest","get_mcp_analysis"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 65, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_chat_signal_quality', '', 'TurnPlan · 多轮 · 测点后问信号质量', '同 session：先测买卖点，再问信号是否靠谱。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我看看中际旭创有没有买卖点」 → 「刚才那个买卖点信号靠谱吗」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"刚才那个买卖点信号靠谱吗","setup_messages":["帮我看看中际旭创有没有买卖点"],"expect_domain":"chat","expect_mode":"talk","expect_sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"],"min_reply_chars":20,"turn_id":"chat_signal_quality","dialogue":[{"role":"user","text":"帮我看看中际旭创有没有买卖点"},{"role":"user","text":"刚才那个买卖点信号靠谱吗","judge":true}],"expect_reply":{"rubric":"在已做买卖点探测后，应对信号是否靠谱做定性说明或限制条件，走闲聊/解释而非再次 probe。"},"expect_intent":{"domain":"chat","mode":"talk","sop":false},"expect_execution":{"profile":"","forbid_tools":["run_strategy_backtest","probe_bot_signal_series"]},"expect_routing":{"domain":"chat","mode":"talk","sop":false,"forbid_tools":["run_strategy_backtest","probe_bot_signal_series"]},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 66, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_reminder_list', '', 'TurnPlan · 单轮 · Reminder 列表', '独立 session：查询 DCA reminder 列表。', '["新 session：运行前清空 Dock Chat","单轮发送：「我现在有哪些 Reminder 提醒」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"我现在有哪些 Reminder 提醒","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_dca_reminders"],"min_reply_chars":20,"turn_id":"bot_reminder_list","dialogue":[{"role":"user","text":"我现在有哪些 Reminder 提醒","judge":true}],"expect_reply":{"rubric":"应列出或说明用户当前的 Reminder/定时提醒，信息结构清晰。","must_cover":["reminder"]},"expect_intent":{"domain":"bot_manage","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["list_dca_reminders"]},"expect_routing":{"domain":"bot_manage","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 67, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_grid_pnl', '', 'TurnPlan · 单轮 · 网格 Bot 盈亏', '独立 session：查询腾讯网格 Bot 盈亏。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我看看腾讯网格 Bot 的盈亏情况」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我看看腾讯网格 Bot 的盈亏情况","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_grid_bots"],"min_reply_chars":20,"turn_id":"bot_grid_pnl","dialogue":[{"role":"user","text":"帮我看看腾讯网格 Bot 的盈亏情况","judge":true}],"expect_reply":{"rubric":"应汇报腾讯网格 Bot 的盈亏或状态，与 grid bot 相关。","must_cover":["网格"]},"expect_intent":{"domain":"bot_manage","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["list_grid_bots"]},"expect_routing":{"domain":"bot_manage","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 68, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_bot_smarttrade_list', '', 'TurnPlan · 单轮 · SmartTrade 列表', '独立 session：列出 SmartTrade 实例。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我查一下我运行中的 SmartTrade 有哪些」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我查一下我运行中的 SmartTrade 有哪些","expect_domain":"bot_manage","expect_mode":"gather","expect_sop":false,"require_tools":["list_smart_trades"],"min_reply_chars":20,"turn_id":"bot_smarttrade_list","dialogue":[{"role":"user","text":"帮我查一下我运行中的 SmartTrade 有哪些","judge":true}],"expect_reply":{"rubric":"应列出 SmartTrade 实例或说明暂无，回应「有哪些 SmartTrade」。","must_cover":["SmartTrade"]},"expect_intent":{"domain":"bot_manage","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["list_smart_trades"]},"expect_routing":{"domain":"bot_manage","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 69, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_backtest_history', '', 'TurnPlan · 多轮 · 回测后查历史', '同 session：先跑回测，再问小米上次回测结果。', '["新 session：运行前清空 Dock Chat","同 session 按序发送：「帮我用SAR加MACD回测一下小米」 → 「小米上次回测结果怎么样」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"小米上次回测结果怎么样","setup_messages":["帮我用SAR加MACD回测一下小米"],"expect_domain":"backtest_history","expect_mode":"gather","expect_sop":false,"require_tools":["list_strategy_backtest_logs"],"min_reply_chars":20,"turn_id":"backtest_history","dialogue":[{"role":"user","text":"帮我用SAR加MACD回测一下小米"},{"role":"user","text":"小米上次回测结果怎么样","judge":true}],"expect_reply":{"rubric":"在刚跑完回测的语境下，应查询或摘要上次/本次回测结果，而非重新发起无关分析。","must_cover":["回测"]},"expect_intent":{"domain":"backtest_history","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["list_strategy_backtest_logs"]},"expect_routing":{"domain":"backtest_history","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 70, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_report_lookup', '', 'TurnPlan · 单轮 · 盘前报告', '独立 session：查询盘前报告内容。', '["新 session：运行前清空 Dock Chat","单轮发送：「今天盘前报告写了什么内容」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"今天盘前报告写了什么内容","expect_domain":"report_lookup","expect_mode":"gather","expect_sop":false,"require_tools":["get_stock_premarket_reports"],"min_reply_chars":20,"turn_id":"report_lookup","dialogue":[{"role":"user","text":"今天盘前报告写了什么内容","judge":true}],"expect_reply":{"rubric":"应返回或摘要今日盘前报告内容，或说明暂无盘前报告。","must_cover":["盘前"]},"expect_intent":{"domain":"report_lookup","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["get_stock_premarket_reports"]},"expect_routing":{"domain":"report_lookup","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 71, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_knowledge_lookup', '', 'TurnPlan · 单轮 · 知识库检索', '独立 session：从知识库讲解 4H MACD。', '["新 session：运行前清空 Dock Chat","单轮发送：「按知识库帮我讲讲4小时MACD怎么用」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"按知识库帮我讲讲4小时MACD怎么用","expect_domain":"knowledge","expect_mode":"gather","expect_sop":false,"require_tools":["search_knowledge"],"min_reply_chars":20,"turn_id":"knowledge_lookup","dialogue":[{"role":"user","text":"按知识库帮我讲讲4小时MACD怎么用","judge":true}],"expect_reply":{"rubric":"应基于知识库讲解 4H MACD，内容偏教学/知识而非直接下单建议。","must_cover":["MACD"]},"expect_intent":{"domain":"knowledge","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["search_knowledge"]},"expect_routing":{"domain":"knowledge","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 72, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_news_lookup', '', 'TurnPlan · 单轮 · 新闻查询', '独立 session：拉取市场新闻。', '["新 session：运行前清空 Dock Chat","单轮发送：「最近有什么财经新闻」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"最近有什么财经新闻","expect_domain":"news","expect_mode":"gather","expect_sop":false,"require_tools":["fetch_market_news"],"min_reply_chars":20,"turn_id":"news_lookup","dialogue":[{"role":"user","text":"最近有什么财经新闻","judge":true}],"expect_reply":{"rubric":"应提供市场或财经新闻摘要，至少列举若干条或说明来源。","must_cover":["新闻"]},"expect_intent":{"domain":"news","mode":"gather","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["fetch_market_news"]},"expect_routing":{"domain":"news","mode":"gather","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 73, 1, datetime('now'), datetime('now'));
INSERT OR IGNORE INTO agent_eval_cases (id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled, created_at, updated_at) VALUES ('turn_plan_dca_grid_backtest', '', 'TurnPlan · 单轮 · DCA 定投回测', '独立 session：DCA/网格类回测意图；若 Agent clarify 标的/参数则自动补默认选项。', '["新 session：运行前清空 Dock Chat","单轮发送：「帮我做一个DCA定投策略回测」","若 Agent clarify：自动回复「用默认定投参数回测腾讯控股」","verify：校验路由/工具/回复关键词 + LLM 语义评判"]', 0, '{"category":"turn_plan","plan_only":false,"session_cleanup":"before_run","dual_model_eval":false,"message":"帮我做一个DCA定投策略回测","expect_domain":"dca_grid","expect_mode":"execute","expect_sop":false,"require_tools":["generate_dca_strategy"],"clarify_reply":"用默认定投参数回测腾讯控股","min_reply_chars":20,"turn_id":"dca_grid_backtest","dialogue":[{"role":"user","text":"帮我做一个DCA定投策略回测"},{"role":"user","text":"用默认定投参数回测腾讯控股","judge":true,"on_clarify":true}],"expect_reply":{"rubric":"应识别 DCA/定投回测意图并给出方案、参数确认或回测执行说明。","must_cover":["DCA"]},"expect_intent":{"domain":"dca_grid","mode":"execute","sop":false},"expect_execution":{"profile":"","legacy_require_tools":["generate_dca_strategy"]},"expect_routing":{"domain":"dca_grid","mode":"execute","sop":false},"judge":{"enabled":true,"model_slot":"auxiliary","min_score":0.7}}', 74, 1, datetime('now'), datetime('now'));
