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
    '{"category":"general","random_stock_enabled":true,"dual_model_eval":false,"session_cleanup":"before_run"}',
    0
) ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order
) VALUES
(
    'strategy_signal_single',
    '',
    '单股 · 单策略 · 信号测试',
    '随机抽取一只股票与一项高级策略，发送自然语言信号测试请求。',
    '["随机选股与策略","发送信号测试请求（含标的与策略名）","校验回复含信号/买卖信息"]',
    TRUE,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"single","stock_count":1,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":80,"pass_keywords":["信号","买","卖"],"session_cleanup":"before_run"}',
    10
),
(
    'strategy_signal_multi_strategy',
    '',
    '单股 · 多策略 · 信号测试',
    '同一只股票上对比 2 项随机策略的信号触发情况。',
    '["随机选股与 2 项策略","发送多策略对比信号测试请求","校验回复含对比/信号摘要"]',
    TRUE,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"multi_strategy","stock_count":1,"strategy_count":2,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["信号","对比"],"session_cleanup":"before_run"}',
    11
),
(
    'strategy_signal_multi_stock',
    '',
    '多股 · 单策略 · 信号测试',
    '同一策略在 2 只随机股票上分别做信号测试并对比。',
    '["随机选 2 只股票与 1 项策略","发送多标的信号测试请求","校验回复含各股信号对比"]',
    TRUE,
    '{"category":"strategy_signal","task":"signal_probe","scenario":"multi_stock","stock_count":2,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["信号","对比"],"session_cleanup":"before_run"}',
    12
),
(
    'strategy_backtest_single',
    '',
    '单股 · 单策略 · 回测',
    '随机单股单策略发送回测请求，校验收益与 log_id。',
    '["随机选股与策略","发送回测请求","校验回复含收益率与 log_id"]',
    TRUE,
    '{"category":"strategy_backtest","task":"backtest","scenario":"single","stock_count":1,"strategy_count":1,"random_stock_enabled":true,"min_reply_chars":80,"pass_keywords":["回测","收益","log"],"session_cleanup":"before_run"}',
    20
),
(
    'strategy_backtest_multi_strategy',
    '',
    '单股 · 多策略 · 回测',
    '同一只股票上对比 2 项随机策略的回测收益。',
    '["随机选股与 2 项策略","发送多策略回测对比请求","校验回复含收益对比与 log_id"]',
    TRUE,
    '{"category":"strategy_backtest","task":"backtest","scenario":"multi_strategy","stock_count":1,"strategy_count":2,"random_stock_enabled":true,"min_reply_chars":100,"pass_keywords":["回测","对比","log"],"session_cleanup":"before_run"}',
    21
),
(
    'strategy_backtest_multi_config',
    '',
    '单股 · 单策略 · 多止盈止损回测',
    '同一策略用两套止盈止损参数（如 5%/3% vs 7%/5%）跑回测并对比。',
    '["随机选股与策略","发送多止盈止损参数回测请求","校验回复含各套配置收益对比"]',
    TRUE,
    '{"category":"strategy_backtest","task":"backtest","scenario":"multi_config","stock_count":1,"strategy_count":1,"config_variants":["止盈5%止损3%","止盈7%止损5%"],"random_stock_enabled":true,"min_reply_chars":120,"pass_keywords":["回测","对比","止盈"],"session_cleanup":"before_run"}',
    22
) ON CONFLICT (id) DO NOTHING;

UPDATE agent_eval_cases SET
    description = '随机抽取一只股票与一项高级策略，发送自然语言信号测试请求。',
    steps_json = '["随机选股与策略","发送信号测试请求（含标的与策略名）","校验回复含信号/买卖信息"]',
    updated_at = NOW()
WHERE id = 'strategy_signal_single';

UPDATE agent_eval_cases SET
    steps_json = '["随机选股与 2 项策略","发送多策略对比信号测试请求","校验回复含对比/信号摘要"]',
    updated_at = NOW()
WHERE id = 'strategy_signal_multi_strategy';

UPDATE agent_eval_cases SET
    description = '随机单股单策略发送回测请求，校验收益与 log_id。',
    updated_at = NOW()
WHERE id = 'strategy_backtest_single';

UPDATE agent_eval_cases SET
    title = '单股 · 单策略 · 多止盈止损回测',
    description = '同一策略用两套止盈止损参数（如 5%/3% vs 7%/5%）跑回测并对比。',
    steps_json = '["随机选股与策略","发送多止盈止损参数回测请求","校验回复含各套配置收益对比"]',
    options_json = '{"category":"strategy_backtest","task":"backtest","scenario":"multi_config","stock_count":1,"strategy_count":1,"config_variants":["止盈5%止损3%","止盈7%止损5%"],"random_stock_enabled":true,"min_reply_chars":120,"pass_keywords":["回测","对比","止盈"],"session_cleanup":"before_run"}',
    updated_at = NOW()
WHERE id = 'strategy_backtest_multi_config';
