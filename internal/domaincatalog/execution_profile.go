package domaincatalog

import (
	"fmt"
	"strings"
)

// ToolScope tells eval where a required tool may appear.
type ToolScope string

const (
	ScopeJudgedTurn      ToolScope = "judged_turn"
	ScopeSession         ToolScope = "session"
	ScopeJudgedOrSession ToolScope = "judged_turn_or_session"
)

// ToolRule is one tool requirement inside an ExecutionProfile.
type ToolRule struct {
	Tool  string
	Scope ToolScope
}

// ExecutionProfile is the canonical execution contract for domain+task patterns.
// Eval and (future) loop gates reference profile IDs — not per-case tool lists.
type ExecutionProfile struct {
	ID                    string
	Required              []ToolRule
	ForbidOnJudgedTurn    []string
	DisallowPriceShortcut bool
}

const (
	ProfileStockPriceSnapshot   = "stock_analysis.price_snapshot"
	ProfileStockPriceViaMCP     = "stock_analysis.price_via_mcp"
	ProfileStockTechnicalFull   = "stock_analysis.technical_full"
	ProfileStockContextFollowup = "stock_analysis.context_followup"
	ProfileStockSymbolResolve   = "stock_analysis.symbol_resolve"
	ProfileSubagentMultiStock     = "subagent.multi_stock_parallel"
)

var executionProfiles = map[string]ExecutionProfile{
	ProfileStockPriceSnapshot: {
		ID: ProfileStockPriceSnapshot,
		Required: []ToolRule{
			{Tool: "search_code", Scope: ScopeJudgedOrSession},
			{Tool: "get_current_price", Scope: ScopeJudgedTurn},
		},
	},
	ProfileStockPriceViaMCP: {
		ID: ProfileStockPriceViaMCP,
		Required: []ToolRule{
			{Tool: "search_code", Scope: ScopeJudgedOrSession},
			{Tool: "get_mcp_analysis", Scope: ScopeJudgedTurn},
		},
		DisallowPriceShortcut: true,
	},
	ProfileStockTechnicalFull: {
		ID: ProfileStockTechnicalFull,
		Required: []ToolRule{
			{Tool: "search_code", Scope: ScopeJudgedOrSession},
			{Tool: "get_mcp_analysis", Scope: ScopeJudgedTurn},
		},
	},
	ProfileStockContextFollowup: {
		ID: ProfileStockContextFollowup,
		Required: []ToolRule{
			{Tool: "search_code", Scope: ScopeSession},
		},
	},
	ProfileStockSymbolResolve: {
		ID: ProfileStockSymbolResolve,
		Required: []ToolRule{
			{Tool: "search_code", Scope: ScopeJudgedTurn},
		},
	},
	ProfileSubagentMultiStock: {
		ID: ProfileSubagentMultiStock,
		Required: []ToolRule{
			{Tool: "delegate_tasks", Scope: ScopeJudgedTurn},
		},
		ForbidOnJudgedTurn: []string{"get_mcp_analysis", "get_current_price", "search_code"},
	},
}

// ExecutionProfileByID returns a registered profile.
func ExecutionProfileByID(id string) (ExecutionProfile, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ExecutionProfile{}, false
	}
	p, ok := executionProfiles[id]
	return p, ok
}

// VerifyExecutionProfile checks judged-turn and session tool traces against a profile.
func VerifyExecutionProfile(profileID string, judgedTools, sessionTools []string) (bool, string) {
	profile, ok := ExecutionProfileByID(profileID)
	if !ok {
		return false, "unknown execution profile " + profileID
	}
	judged := normalizeToolSet(judgedTools)
	session := normalizeToolSet(sessionTools)
	union := unionToolSets(judged, session)

	var problems []string
	for _, rule := range profile.Required {
		if !ruleSatisfied(rule, judged, session, union) {
			problems = append(problems, fmt.Sprintf("missing tool %s (%s)", rule.Tool, rule.Scope))
		}
	}
	for _, tool := range profile.ForbidOnJudgedTurn {
		if judged[tool] {
			problems = append(problems, fmt.Sprintf("forbid tool %s on judged turn", tool))
		}
	}
	if profile.DisallowPriceShortcut {
		if judged["get_current_price"] && !judged["get_mcp_analysis"] {
			problems = append(problems, "price shortcut: get_current_price without get_mcp_analysis on judged turn")
		}
	}
	if len(problems) > 0 {
		return false, strings.Join(problems, "; ")
	}
	return true, fmt.Sprintf("profile=%s judged=%s session=%s", profileID, toolNames(judged), toolNames(session))
}

func ruleSatisfied(rule ToolRule, judged, session, union map[string]bool) bool {
	switch rule.Scope {
	case ScopeJudgedTurn:
		return judged[rule.Tool]
	case ScopeSession:
		return session[rule.Tool]
	case ScopeJudgedOrSession:
		return union[rule.Tool]
	default:
		return union[rule.Tool]
	}
}

func normalizeToolSet(tools []string) map[string]bool {
	out := map[string]bool{}
	for _, t := range tools {
		t = strings.TrimSpace(t)
		if t != "" {
			out[t] = true
		}
	}
	return out
}

func unionToolSets(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range a {
		if v {
			out[k] = true
		}
	}
	for k, v := range b {
		if v {
			out[k] = true
		}
	}
	return out
}

func toolNames(set map[string]bool) string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	return strings.Join(names, ",")
}
