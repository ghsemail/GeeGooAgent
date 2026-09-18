package eval

// IndividualWorkflowEvalCases returns live Dock Chat eval cases for chat workflows
// (multi-step serial execution, not TurnPlan routing regression).
func IndividualWorkflowEvalCases() []TurnPlanEvalCaseDef {
	return []TurnPlanEvalCaseDef{
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
