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
			Title:       "legacy open-flow · Workflow · 生成策略档案（Macd4H）",
			Description: "legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner（#106 默认 Chat Loop 未开 open-flow）。generate_strategy_archive：读策略库 → LLM 合成策略档案 → 写入并读回 WeKnora 知识库。",
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
			Title:       "legacy open-flow · Workflow · 读取策略（Macd4H）",
			Description: "legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。strategy_dev：读取 Macd4H 策略档案（无则自动生成），注入开发上下文。",
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
			Title:       "legacy open-flow · Workflow · 多策略信号对比",
			Description: "legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。固定标的，Workflow 串行 probe 多策略并输出对比表（非 ReAct）。",
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
			Description: "Chat Loop（#106）：skill catalog → 可选 skill_view → 常规 probe/诊断工具 → 可用 SAR·腾讯 诊断回复（workflow_card / Episode 非硬性要求）。",
			Steps: []string{
				"发送策略开发面板同款信号诊断请求（含 workflow_options）",
				"期望经 catalog 与普通工具完成 probe 与诊断",
				"校验回复含 SAR、腾讯及诊断/命中/触发类结论",
				"校验不应触发用户 clarify（标的与信号已在 message/options 中）",
			},
			SortOrder: 14,
			Options: TurnPlanCaseOptions{
				Category:       WorkflowCatID,
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 信号作为卖出信号，开启阻力支撑熔断，测试一下腾讯控股（00700.HK）。",
				MinReplyChars:  120,
				PassKeywords:   []string{"SAR", "腾讯", "诊断", "命中", "触发"},
				WaitTimeoutSec: 300,
				WorkflowOptions: map[string]any{
					"signal_diagnose": map[string]any{
						"panel_strategy_label": "买:SAR · 卖:SAR",
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
			Title:       "legacy open-flow · Workflow · 生成策略档案（随机策略）",
			Description: "legacy open-flow：仍依赖 run_chat_workflow_turn / Workflow Runner。generate_strategy_archive：从策略库随机选一项，LLM 合成策略档案并写入知识库。",
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
