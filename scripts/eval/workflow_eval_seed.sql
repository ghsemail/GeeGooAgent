DELETE FROM agent_eval_cases WHERE id IN ('taskflow_multi_strategy_compare', 'workflow_strategy_dev_cognition');

INSERT INTO agent_eval_cases (
    id, user_id, title, description, steps_json, supports_random_stock, options_json, sort_order, enabled
) VALUES (
    'workflow_generate_strategy_cognition',
    '',
    'legacy open-flow · Workflow · 生成策略档案（Macd4H）',
    'legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner（#106 默认 Chat Loop 未开 open-flow）。generate_strategy_archive：读策略库 → LLM 合成策略档案 → 写入并读回 WeKnora 知识库。',
    '["发送生成策略档案请求（Macd4H）","校验 workflow 完成档案生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_archive","scenario":"cognition","workflow_skill":"generate_strategy_archive","random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略档案","知识库","策略库"],"session_cleanup":"before_run","message":"帮我生成 Macd4H 的策略档案","wait_timeout_sec":900}',
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
    'legacy open-flow · Workflow · 读取策略（Macd4H）',
    'legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。strategy_dev：读取 Macd4H 策略档案（无则自动生成），注入开发上下文。',
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
    'legacy open-flow · Workflow · 多策略信号对比',
    'legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。固定标的，Workflow 串行 probe 多策略并输出对比表（非 ReAct）。',
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
    'Chat Loop（#106）：skill catalog → 可选 skill_view → 常规 probe/诊断工具 → 可用 SAR·腾讯 诊断回复（workflow_card / Episode 非硬性要求）。',
    '["发送策略开发面板同款信号诊断请求（含 workflow_options）","期望经 catalog 与普通工具完成 probe 与诊断","校验回复含 SAR、腾讯及诊断/命中/触发类结论","校验不应触发用户 clarify"]',
    FALSE,
    '{"category":"workflow","task":"signal_diagnose","scenario":"fixed_dialogue","workflow_skill":"signal_diagnose","random_stock_enabled":false,"min_reply_chars":120,"pass_keywords":["SAR","腾讯","诊断","命中","触发"],"session_cleanup":"before_run","message":"信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 信号作为卖出信号，开启阻力支撑熔断，测试一下腾讯控股（00700.HK）。","wait_timeout_sec":300,"workflow_options":{"signal_diagnose":{"panel_strategy_label":"买:SAR · 卖:SAR","frequency":"60m","months_back":3,"buy_signal":[{"index":"SAR","type":"signal"}],"sell_signal":[{"index":"SAR","type":"signal"}]}}}',
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
    'legacy open-flow · Workflow · 生成策略档案（随机策略）',
    'legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。generate_strategy_archive：从策略库随机选一项，LLM 合成策略档案并写入知识库。',
    '["随机选取一项 catalog 组合策略","发送生成策略档案请求","校验 workflow 完成档案生成报告","校验回复含策略库/知识库与读回验证"]',
    FALSE,
    '{"category":"workflow","task":"generate_strategy_archive","scenario":"cognition_random","workflow_skill":"generate_strategy_archive","random_strategy_enabled":true,"random_stock_enabled":false,"min_reply_chars":80,"pass_keywords":["策略档案","知识库","策略库"],"session_cleanup":"before_run","message":"帮我生成策略档案","wait_timeout_sec":900}',
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
