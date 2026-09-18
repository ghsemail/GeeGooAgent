DELETE FROM agent_eval_cases WHERE id IN ('taskflow_multi_strategy_compare', 'workflow_strategy_dev_cognition');

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_generate_strategy_cognition',
    '',
    'Workflow · 生成策略认知（Macd4H）',
    'generate_strategy_cognition：读策略库 → LLM 合成 Agent 认知 → 写入并读回 WeKnora 知识库。',
    '["发送生成策略认知请求（Macd4H）","校验 workflow 完成认知生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_cognition","scenario":"cognition","workflow_skill":"generate_strategy_cognition","random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略认知","Macd4H","知识库","策略库"],"session_cleanup":"before_run","message":"生成策略认知 Macd4H","wait_timeout_sec":900}',
    10,
    TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json,
    options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_strategy_dev_read_cognition',
    '',
    'Workflow · 策略开发读认知（Macd4H）',
    'strategy_dev：从知识库读取 Macd4H 策略认知作为开发上下文。',
    '["发送策略开发请求（Macd4H）","校验 workflow 已加载知识库策略认知"]',
    FALSE,
    '{"category":"workflow","task":"strategy_dev","scenario":"read_cognition","workflow_skill":"strategy_dev","random_stock_enabled":false,"min_reply_chars":60,"pass_keywords":["策略开发","Macd4H","知识库","策略认知"],"session_cleanup":"before_run","message":"策略开发 Macd4H"}',
    12,
    TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json,
    options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_multi_strategy_compare',
    '',
    'Workflow · 多策略信号对比',
    '固定标的，Workflow 串行 probe 多策略并输出对比表（非 ReAct）。',
    '["发送多策略对比请求","校验 workflow 对比表与买/卖次","校验回复含多策略信号对比"]',
    FALSE,
    '{"category":"workflow","task":"multi_strategy_compare","scenario":"fixed_dialogue","stock_count":1,"strategy_count":2,"random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["多策略","对比","买","卖"],"session_cleanup":"before_run","message":"帮我在腾讯上对比 Macd4H 和 共振的信号买卖点"}',
    11,
    TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json,
    options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_generate_strategy_cognition_random',
    '',
    'Workflow · 生成策略认知（随机策略）',
    'generate_strategy_cognition：从策略库随机选一项，LLM 合成认知并写入知识库。',
    '["随机选取一项 catalog 组合策略","发送生成策略认知请求","校验 workflow 完成认知生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_cognition","scenario":"cognition_random","workflow_skill":"generate_strategy_cognition","random_strategy_enabled":true,"random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略认知","知识库","策略库"],"session_cleanup":"before_run","message":"生成策略认知","wait_timeout_sec":900}',
    13,
    TRUE
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    steps_json = EXCLUDED.steps_json,
    options_json = EXCLUDED.options_json,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    updated_at = NOW();
