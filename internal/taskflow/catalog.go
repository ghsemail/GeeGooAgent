package taskflow

// TemplateInfo describes one predefined serial task flow exposed on the Agent platform.
type TemplateInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"` // available | planned
	Triggers    []string `json:"triggers,omitempty"`
	Phases      []string `json:"phases,omitempty"`
	ResumeHints []string `json:"resume_hints,omitempty"`
}

// ListTemplates returns the built-in TaskFlow catalog (SSOT for dashboard + docs).
func ListTemplates() []TemplateInfo {
	return []TemplateInfo{
		{
			ID:          TemplateMultiStrategyCompare,
			Name:        "多策略信号对比",
			Description: "固定标的，串行 probe 多个策略并输出买/卖次对比表。适用于「挨个跑」「多策略对比」等场景。",
			Status:      "available",
			Triggers: []string{
				"多策略", "对比", "挨个跑", "依次测", "哪个信号多",
			},
			Phases: []string{
				PhaseResolveSymbol,
				PhasePickStrategies,
				PhaseProbeForeach,
				PhaseSummarize,
			},
			ResumeHints: []string{"继续", "重试失败", "POST /v1/chat/flow/resume"},
		},
		{
			ID:          "param_tune",
			Name:        "策略参数调优",
			Description: "固定标的与策略，串行尝试 months_back / frequency 等参数组合，改善买卖点可见性。",
			Status:      "planned",
			Triggers:    []string{"买卖点不明显", "信号太少", "帮我调参"},
			Phases: []string{
				"resolve_context",
				"baseline_probe",
				"pick_param_variants",
				"probe_foreach",
				PhaseSummarize,
			},
		},
	}
}

// ListTemplatesPayload returns dashboard/API-friendly maps.
func ListTemplatesPayload() []map[string]any {
	out := make([]map[string]any, 0, len(ListTemplates()))
	for _, t := range ListTemplates() {
		out = append(out, map[string]any{
			"id":           t.ID,
			"name":         t.Name,
			"description":  t.Description,
			"status":       t.Status,
			"triggers":     t.Triggers,
			"phases":       t.Phases,
			"resume_hints": t.ResumeHints,
			"kind":         "taskflow",
		})
	}
	return out
}

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
	return map[string]any{
		"run_id":       flow.RunID,
		"template":     flow.Template,
		"template_name": templateName(flow.Template),
		"phase":        flow.Phase,
		"status":       flow.Status,
		"stock_query":  flow.StockQuery,
		"stock_code":   flow.StockCode,
		"stock_name":   flow.StockName,
		"months_back":  flow.MonthsBack,
		"cursor":       flow.Cursor,
		"total":        total,
		"done":         done,
		"strategies":   strategies,
		"partial_report": flow.PartialReport,
	}
}

func templateName(id string) string {
	for _, t := range ListTemplates() {
		if t.ID == id {
			return t.Name
		}
	}
	return id
}
