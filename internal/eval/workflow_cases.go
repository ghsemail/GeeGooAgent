package eval

// IndividualWorkflowEvalCases returns live Dock Chat eval cases for chat workflows
// (multi-step serial execution, not TurnPlan routing regression).
func IndividualWorkflowEvalCases() []TurnPlanEvalCaseDef {
	return []TurnPlanEvalCaseDef{
		{
			ID:          "workflow_generate_strategy_cognition",
			Title:       "Workflow · 生成策略认知（Macd4H）",
			Description: "generate_strategy_cognition：读策略库 → LLM 合成 Agent 认知 → 写入并读回 WeKnora 知识库。",
			Steps: []string{
				"发送生成策略认知请求（Macd4H）",
				"校验 workflow 完成认知生成报告",
				"校验回复含策略库/知识库与读回验证",
			},
			SortOrder: 10,
			Options: TurnPlanCaseOptions{
				Category:       "workflow",
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "生成策略认知 Macd4H",
				MinReplyChars:  80,
				PassKeywords:   []string{"策略认知", "Macd4H", "知识库", "策略库"},
				WaitTimeoutSec: 900,
			},
		},
		{
			ID:          "workflow_strategy_dev_read_cognition",
			Title:       "Workflow · 策略开发读认知（Macd4H）",
			Description: "strategy_dev：从知识库读取 Macd4H 策略认知作为开发上下文。",
			Steps: []string{
				"发送策略开发请求（Macd4H）",
				"校验 workflow 已加载知识库策略认知",
			},
			SortOrder: 12,
			Options: TurnPlanCaseOptions{
				Category:       "workflow",
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "策略开发 Macd4H",
				MinReplyChars:  60,
				PassKeywords:   []string{"策略开发", "Macd4H", "知识库", "策略认知"},
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
				Category:       "workflow",
				SessionCleanup: DefaultEvalSessionCleanup,
				Message:        "帮我在腾讯上对比 Macd4H 和 共振的信号买卖点",
				MinReplyChars:  80,
				PassKeywords:   []string{"多策略", "对比", "买", "卖"},
			},
		},
	}
}
