package skills

// Chat-triggered workflow skill names (must match internal/workflow/chat/types.go).
const (
	SkillMultiStrategyCompare      = "multi_strategy_compare"
	SkillParamTune                 = "param_tune"
	SkillStrategyDev               = "strategy_dev"
	SkillGenerateStrategyCognition = "generate_strategy_cognition"
)

// ChatTrigger describes chat keyword entry for a workflow skill.
type ChatTrigger struct {
	Status      string   // available | planned
	Triggers    []string
	Phases      []string
	ResumeHints []string
}

// ChatWorkflowCatalog returns dashboard metadata for chat-triggered workflow skills.
func ChatWorkflowCatalog() []map[string]any {
	return []map[string]any{
		{
			"id":            SkillGenerateStrategyCognition,
			"name":          "生成策略档案",
			"description":   "读策略库 → LLM 合成策略档案 → 写入并读回 WeKnora 知识库。",
			"status":        "available",
			"triggers":      []string{"生成策略档案", "策略档案", "学习策略", "了解策略", "写入知识库"},
			"phases":        []string{"cognition_pick", "cognition_read_catalog", "cognition_compose", "cognition_save_kb", "cognition_verify_kb", "summarize"},
			"resume_hints":  []string{"继续", "重试失败", "POST /v1/chat/workflow/resume"},
			"kind":          "workflow",
			"trigger_modes": []string{"chat"},
		},
		{
			"id":            SkillStrategyDev,
			"name":          "策略开发",
			"description":   "确保策略档案存在（无则自动生成），读取并注入策略开发上下文（后续 Step 扩展回测/调参等）。",
			"status":        "available",
			"triggers":      []string{"读取", "策略开发"},
			"phases":        []string{"dev_pick", "dev_ensure_archive", "summarize"},
			"resume_hints":  []string{"继续", "重试失败", "POST /v1/chat/workflow/resume"},
			"kind":          "workflow",
			"trigger_modes": []string{"chat"},
		},
		{
			"id":           SkillMultiStrategyCompare,
			"name":         "多策略信号对比",
			"description":  "固定标的，串行 probe 多个策略并输出买/卖次对比表。适用于「挨个跑」「多策略对比」等场景。",
			"status":       "available",
			"triggers":     []string{"多策略", "对比", "挨个跑", "依次测", "哪个信号多"},
			"phases":       []string{"resolve_symbol", "pick_strategies", "probe_foreach", "summarize"},
			"resume_hints": []string{"继续", "重试失败", "POST /v1/chat/workflow/resume"},
			"kind":         "workflow",
			"trigger_modes": []string{"chat", "cron"},
		},
		{
			"id":          SkillParamTune,
			"name":        "策略参数调优",
			"description": "固定标的与策略，串行尝试 months_back / frequency 等参数组合，改善买卖点可见性。",
			"status":      "planned",
			"triggers":    []string{"买卖点不明显", "信号太少", "帮我调参"},
			"phases": []string{
				"resolve_context", "baseline_probe", "pick_param_variants", "probe_foreach", "summarize",
			},
			"kind":          "workflow",
			"trigger_modes": []string{"chat", "cron"},
		},
	}
}

// IsChatWorkflowSkill reports whether name is a chat+cron workflow (not L5 batch-only).
func IsChatWorkflowSkill(name string) bool {
	switch name {
	case SkillMultiStrategyCompare, SkillParamTune, SkillStrategyDev, SkillGenerateStrategyCognition:
		return true
	default:
		return false
	}
}

// ChatTriggerFor returns chat trigger metadata for a registered skill.
func ChatTriggerFor(spec Spec) *ChatTrigger {
	return spec.Chat
}
