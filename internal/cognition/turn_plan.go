package cognition

import (
	"context"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// Domain is the chat capability bucket for one turn.
type Domain string

const (
	DomainChat            Domain = "chat"
	DomainStockAnalysis   Domain = "stock_analysis"
	DomainNews            Domain = "news"
	DomainKnowledge       Domain = "knowledge"
	DomainReportLookup    Domain = "report_lookup"
	DomainReportWrite     Domain = "report_write"
	DomainBotManage       Domain = "bot_manage"
	DomainSignalProbe     Domain = "signal_probe"
	DomainBacktestRun     Domain = "backtest_run"
	DomainBacktestHistory Domain = "backtest_history"
	DomainCustomSignal    Domain = "custom_signal"
	DomainPromptAdmin     Domain = "prompt_admin"
	DomainDCAGrid         Domain = "dca_grid"
	DomainAmbiguous       Domain = "ambiguous"
)

// Mode is how the kernel should spend the turn.
type Mode string

const (
	ModeTalk    Mode = "talk"
	ModeGather  Mode = "gather"
	ModeClarify Mode = "clarify"
	ModeExecute Mode = "execute"
)

// TurnPlan is the per-turn routing decision: what the user wants, which
// skills/tools may run, and whether to execute a playbook.
type TurnPlan struct {
	Domain          Domain   `json:"domain"`
	Act             string   `json:"act"`
	Mode            Mode     `json:"mode"`
	Confidence      float64  `json:"confidence"`
	Reason          string   `json:"reason"`
	Skills          []string `json:"skills,omitempty"`
	ToolsAllow      []string `json:"tools_allow,omitempty"`
	ClarifyQuestion string   `json:"clarify_question,omitempty"`
	ClarifyChoices  []string `json:"clarify_choices,omitempty"`
}

// PlanInput is the planner view of the current user turn.
type PlanInput struct {
	Ctx        context.Context
	UserText   string
	LastDomain Domain
}

// Planner classifies a user turn before skills are injected or tools run.
type Planner interface {
	Plan(in PlanInput) TurnPlan
}

// ShouldRunBacktestPlaybook reports whether the deterministic backtest SOP may run.
func (p TurnPlan) ShouldRunBacktestPlaybook() bool {
	return p.Domain == DomainBacktestRun && p.Mode == ModeExecute
}

// FilterSchemas keeps only tools allowed by the plan (plus clarify).
// If the incoming list is already a subset (tests pass a single tool), the
// intersection with ToolsAllow is applied; unknown provided tools are dropped.
func FilterSchemas(schemas []llm.ToolSchema, plan TurnPlan) []llm.ToolSchema {
	if len(schemas) == 0 {
		return schemas
	}
	allow := map[string]struct{}{alwaysAllowClarify: {}}
	for _, name := range plan.ToolsAllow {
		name = strings.TrimSpace(name)
		if name != "" {
			allow[name] = struct{}{}
		}
	}
	out := make([]llm.ToolSchema, 0, len(schemas))
	for _, s := range schemas {
		if _, ok := allow[s.Name]; ok {
			out = append(out, s)
		}
	}
	return out
}

const alwaysAllowClarify = "clarify"
