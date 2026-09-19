package chat

// CardPayload builds SSE card data for the live UI.
func CardPayload(flow *Flow) map[string]any {
	if flow == nil {
		return nil
	}
	strategies := make([]map[string]any, 0, len(flow.Strategies))
	done, total := 0, len(flow.Strategies)
	for _, s := range flow.Strategies {
		if s.Status == StepDone || s.Status == StepSkipped {
			done++
		}
		strategies = append(strategies, map[string]any{
			"query":      s.Query,
			"label":      s.Label,
			"frequency":  s.Frequency,
			"status":     s.Status,
			"buy_hits":   s.BuyHits,
			"sell_hits":  s.SellHits,
			"last_error": s.LastError,
		})
	}
	skill := flow.Template
	out := map[string]any{
		"run_id":         flow.RunID,
		"skill":          skill,
		"template":       skill, // legacy clients
		"template_name":  skillDisplayName(skill),
		"phase":          flow.Phase,
		"status":         flow.Status,
		"stock_query":    flow.StockQuery,
		"stock_code":     flow.StockCode,
		"stock_name":     flow.StockName,
		"months_back":    flow.MonthsBack,
		"cursor":         flow.Cursor,
		"total":          total,
		"done":           done,
		"strategies":     strategies,
		"partial_report": flow.PartialReport,
	}
	if canonicalSkill(skill) == SkillGenerateStrategyArchive || skill == SkillStrategyDev {
		out["workflow_step"] = flow.WorkflowStep
		out["strategy_query"] = flow.StrategyQuery
		out["catalog_type"] = flow.CatalogType
		out["catalog_label"] = flow.CatalogLabel
		out["knowledge_id"] = flow.KnowledgeID
		out["kb_draft_chars"] = len(flow.KBDraft)
	}
	if canonicalSkill(skill) == SkillSignalDiagnose {
		out["strategy_query"] = flow.StrategyQuery
		out["catalog_type"] = flow.CatalogType
		out["catalog_label"] = flow.CatalogLabel
		out["knowledge_id"] = flow.KnowledgeID
		out["kb_draft_chars"] = len(flow.KBDraft)
		out["eval_judgment"] = flow.EvalJudgment
	}
	return out
}

func skillDisplayName(id string) string {
	switch id {
	case "multi_strategy_compare":
		return "多策略信号对比"
	case "param_tune":
		return "策略参数调优"
	case "strategy_dev":
		return "策略开发"
	case SkillSignalDiagnose:
		return "信号诊断"
	case SkillGenerateStrategyArchive, legacyGenerateStrategyCognition:
		return "生成策略档案"
	default:
		return id
	}
}
