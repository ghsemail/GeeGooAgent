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
	return map[string]any{
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
}

func skillDisplayName(id string) string {
	switch id {
	case "multi_strategy_compare":
		return "多策略信号对比"
	case "param_tune":
		return "策略参数调优"
	default:
		return id
	}
}
