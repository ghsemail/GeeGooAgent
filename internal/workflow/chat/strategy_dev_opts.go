package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// ApplyPendingStrategyDevOpts copies session pending opts onto a new pipeline flow and clears pending.
func ApplyPendingStrategyDevOpts(session *runtime.Session, flow *Flow) {
	if session == nil || flow == nil || session.PendingStrategyDevOpts == nil {
		return
	}
	o := session.PendingStrategyDevOpts
	if strings.TrimSpace(o.Code) != "" {
		flow.StockCode = strings.TrimSpace(o.Code)
		flow.StockQuery = flow.StockCode
	}
	if strings.TrimSpace(o.Frequency) != "" {
		flow.ProbeFrequencyOverride = strings.TrimSpace(o.Frequency)
	}
	if o.MonthsBack > 0 {
		flow.MonthsBack = o.MonthsBack
	}
	session.PendingStrategyDevOpts = nil
}

// ParseStrategyDevOptsFromWorkflowOptions reads workflow_options.strategy_dev from chat request.
func ParseStrategyDevOptsFromWorkflowOptions(raw map[string]any) *runtime.StrategyDevOpts {
	if len(raw) == 0 {
		return nil
	}
	block, _ := raw["strategy_dev"].(map[string]any)
	if block == nil {
		return nil
	}
	out := &runtime.StrategyDevOpts{}
	if s, ok := block["code"].(string); ok {
		out.Code = strings.TrimSpace(s)
	}
	if s, ok := block["frequency"].(string); ok {
		out.Frequency = strings.TrimSpace(s)
	}
	if v, ok := block["months_back"].(float64); ok && v > 0 {
		out.MonthsBack = int(v)
	} else if v, ok := block["months_back"].(int); ok && v > 0 {
		out.MonthsBack = v
	}
	if out.Code == "" && out.Frequency == "" && out.MonthsBack == 0 {
		return nil
	}
	return out
}
