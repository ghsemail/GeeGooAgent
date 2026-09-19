DELETE FROM agent_eval_cases WHERE id IN ('taskflow_multi_strategy_compare', 'workflow_strategy_dev_cognition');

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_generate_strategy_cognition',
    '',
    'Workflow · 生成策略档案（Macd4H）',
    'generate_strategy_cognition：读策略库 → LLM 合成策略档案 → 写入并读回 WeKnora 知识库。',
    '["发送生成策略档案请求（Macd4H）","校验 workflow 完成档案生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_cognition","scenario":"cognition","workflow_skill":"generate_strategy_cognition","random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略档案","知识库","策略库"],"session_cleanup":"before_run","message":"帮我生成 Macd4H 的策略档案","wait_timeout_sec":900}',
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
    'Workflow · 读取策略（Macd4H）',
    'strategy_dev：读取 Macd4H 策略档案（无则自动生成），注入开发上下文。',
    '["发送读取策略请求（Macd4H）","校验 workflow 已加载或生成策略档案"]',
    FALSE,
    '{"category":"workflow","task":"strategy_dev","scenario":"read_cognition","workflow_skill":"strategy_dev","random_stock_enabled":false,"min_reply_chars":60,"pass_keywords":["读取","Macd4H","知识库","策略档案"],"session_cleanup":"before_run","message":"读取 Macd4H 策略"}',
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
    'workflow_signal_diagnose_sar_tencent',
    '',
    'Workflow · 信号诊断（SAR · 腾讯）',
    'signal_diagnose：probe → Episode 命中率评价 → LLM 诊断报告。',
    '["发送信号诊断请求（SAR · 腾讯）","校验 workflow 含 Episode 命中率与诊断结论","校验回复含 SAR / 腾讯 / 命中"]',
    FALSE,
    '{"category":"workflow","task":"signal_diagnose","scenario":"fixed_dialogue","workflow_skill":"signal_diagnose","random_stock_enabled":false,"min_reply_chars":120,"pass_keywords":["Episode","命中","SAR","腾讯"],"session_cleanup":"before_run","message":"诊断 SAR · 腾讯","wait_timeout_sec":300}',
    14,
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
    'Workflow · 生成策略档案（随机策略）',
    'generate_strategy_cognition：从策略库随机选一项，LLM 合成策略档案并写入知识库。',
    '["随机选取一项 catalog 组合策略","发送生成策略档案请求","校验 workflow 完成档案生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_cognition","scenario":"cognition_random","workflow_skill":"generate_strategy_cognition","random_strategy_enabled":true,"random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略档案","知识库","策略库"],"session_cleanup":"before_run","message":"帮我生成策略档案","wait_timeout_sec":900}',
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
