package eval

const WorkflowCatID = "workflow"

// WorkflowCategories groups workflow live eval cases for dashboard / automation.
func WorkflowCategories() []TurnPlanCategory {
	return []TurnPlanCategory{
		{ID: WorkflowCatID, Title: "Workflow", Order: 9},
	}
}

// EvalAutoCategories is the canonical category order for auto eval (TurnPlan + Workflow).
func EvalAutoCategories() []TurnPlanCategory {
	out := append([]TurnPlanCategory{}, TurnPlanCategories()...)
	out = append(out, WorkflowCategories()...)
	return out
}

// IndividualWorkflowEvalCases returns live Dock Chat eval cases for chat workflows
// (multi-step serial execution, not TurnPlan routing regression).
func IndividualWorkflowEvalCases() []TurnPlanEvalCaseDef {
	return []TurnPlanEvalCaseDef{
		{
			ID:          "workflow_generate_strategy_cognition",
			Title:       "Workflow · 生成策略档案（Macd4H）",
			Description: "generate_strategy_archive：读策略库 → LLM 合成策略档案 → 写入并读回 WeKnora 知识库。",
			Steps: []string{
				"发送生成策略档案请求（Macd4H）",
				"校验 workflow 完成档案生成报告",
				"校验回复含策略库/知识库与读回验证",
			},
			SortOrder: 10,
			Options: TurnPlanCaseOptions{
				Category:       WorkflowCatID,
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "帮我生成 Macd4H 的策略档案",
				MinReplyChars:  80,
				PassKeywords:   []string{"策略档案", "知识库", "策略库"},
				WaitTimeoutSec: 900,
			},
		},
		{
			ID:          "workflow_strategy_dev_read_cognition",
			Title:       "Workflow · 读取策略（Macd4H）",
			Description: "strategy_dev：读取 Macd4H 策略档案（无则自动生成），注入开发上下文。",
			Steps: []string{
				"发送读取策略请求（Macd4H）",
				"校验 workflow 已加载或生成策略档案",
			},
			SortOrder: 12,
			Options: TurnPlanCaseOptions{
				Category:       WorkflowCatID,
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "读取 Macd4H 策略",
				MinReplyChars:  60,
				PassKeywords:   []string{"读取", "Macd4H", "知识库", "策略档案"},
			},
		},
		{
			ID:          "workflow_multi_strategy_compare",
			Title:       "Workflow · 多策略信号对比",
			Description: "固定标的，Workflow 串行 probe 多策略并输出对比表（非 ReAct）。",
			Steps: []string{
				"发送多策略对比请求",
				"校验 workflow 对比表与买/卖次",
				"校验回复含多策略信号对比",
			},
			SortOrder: 11,
			Options: TurnPlanCaseOptions{
				Category:       WorkflowCatID,
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "帮我在腾讯上对比 Macd4H 和 共振的信号买卖点",
				MinReplyChars:  80,
				PassKeywords:   []string{"多策略", "对比", "买", "卖"},
			},
		},
		{
			ID:          "workflow_signal_diagnose_sar_tencent",
			Title:       "Workflow · 信号诊断（SAR · 腾讯）",
			Description: "signal_diagnose：probe（含 key_levels）→ Episode 命中率评价 → LLM 诊断报告。",
			Steps: []string{
				"发送策略开发面板同款组合式信号诊断请求（买卖 SAR · 腾讯 · 含 workflow_options）",
				"校验 workflow 含 Episode 命中率与诊断结论",
				"校验回复含 SAR / 腾讯 / 命中",
				"校验不应触发用户 clarify（标的与信号已在 message/options 中）",
			},
			SortOrder: 14,
			Options: TurnPlanCaseOptions{
				Category:       WorkflowCatID,
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 信号作为卖出信号，开启阻力支撑熔断，测试一下腾讯控股（00700.HK）。",
				MinReplyChars:  120,
				PassKeywords:   []string{"Episode", "命中", "SAR", "腾讯"},
				WaitTimeoutSec: 300,
				WorkflowOptions: map[string]any{
					"signal_diagnose": map[string]any{
						"use_key_level_episode_stop": true,
						"key_break_mode":             "resist_high",
						"panel_strategy_label":       "买:SAR · 卖:SAR",
						"frequency":                  "60m",
						"months_back":                3,
						"buy_signal": []any{
							map[string]any{"index": "SAR", "type": "signal"},
						},
						"sell_signal": []any{
							map[string]any{"index": "SAR", "type": "signal"},
						},
					},
				},
			},
		},
		{
			ID:          "workflow_generate_strategy_cognition_random",
			Title:       "Workflow · 生成策略档案（随机策略）",
			Description: "generate_strategy_archive：从策略库随机选一项，LLM 合成策略档案并写入知识库。",
			Steps: []string{
				"随机选取一项 catalog 组合策略",
				"发送生成策略档案请求",
				"校验 workflow 完成档案生成报告",
				"校验回复含策略库/知识库与读回验证",
			},
			SortOrder: 13,
			Options: TurnPlanCaseOptions{
				Category:              WorkflowCatID,
				SessionCleanup:        DefaultEvalSessionCleanup,
				Message:               "帮我生成策略档案",
				RandomStrategyEnabled: true,
				MinReplyChars:         80,
				PassKeywords:          []string{"策略档案", "知识库", "策略库"},
				WaitTimeoutSec:        900,
			},
		},
	}
}
