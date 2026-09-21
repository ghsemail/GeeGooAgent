package eval

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// PresetClarifyAnswerFromSession returns the user text injected after preset clarify (2nd user message).
func PresetClarifyAnswerFromSession(chat *chatsession.ChatSession) string {
	if chat == nil {
		return ""
	}
	userIdx := 0
	for _, msg := range chat.Messages {
		if msg.Role != llm.RoleUser {
			continue
		}
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			continue
		}
		userIdx++
		if userIdx >= 2 {
			return text
		}
	}
	for i := len(chat.StepRecords) - 1; i >= 0; i-- {
		rec := chat.StepRecords[i]
		if rec.Kind != "clarify" {
			continue
		}
		s := strings.TrimSpace(rec.Summary)
		if strings.HasPrefix(s, "answer=") {
			return strings.TrimSpace(strings.TrimPrefix(s, "answer="))
		}
	}
	return ""
}

func choiceIndicatesPriceSnapshot(choice string) bool {
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice == "" {
		return false
	}
	for _, token := range []string{"现价", "当前价", "报价", "只要价", "只看价"} {
		if strings.Contains(choice, token) {
			return true
		}
	}
	return false
}

func choiceIndicatesAnalysis(choice string) bool {
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice == "" {
		return false
	}
	for _, token := range []string{"分析", "走势", "技术面", "深度"} {
		if strings.Contains(choice, token) {
			return true
		}
	}
	return false
}

func choiceIndicatesAnalysisOnly(choice string) bool {
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return false
	}
	return strings.Contains(choice, "先只做分析") ||
		(strings.Contains(choice, "分析") && !strings.Contains(choice, "回测"))
}

// UsesFirstTurnIntentVerify reports single-turn ambiguous/clarify cases (routing checked on turn 1).
func UsesFirstTurnIntentVerify(opts TurnPlanCaseOptions) bool {
	intent := opts.Normalize().intent()
	if !strings.EqualFold(intent.Domain, "ambiguous") || !strings.EqualFold(intent.Mode, "clarify") {
		return false
	}
	regular, _ := DialogueExecutionPlan(opts.Normalize())
	return len(regular) <= 1
}

func executionSpecForClarifyBranch(chat *chatsession.ChatSession, opts TurnPlanCaseOptions, spec ExpectExecutionSpec) ExpectExecutionSpec {
	choice := PresetClarifyAnswerFromSession(chat)
	out := spec
	out.ForbidTools = append([]string(nil), spec.ForbidTools...)

	switch strings.TrimSpace(opts.TurnID) {
	case "stock_quote_ambiguous":
		out.Profile = ""
		out.ForbidTools = []string{"run_strategy_backtest"}
		switch {
		case choiceIndicatesPriceSnapshot(choice):
			out.Profile = "stock_analysis.price_snapshot"
			out.ForbidTools = append(out.ForbidTools, "get_mcp_analysis")
		case choiceIndicatesAnalysis(choice):
			out.ForbidTools = []string{"run_strategy_backtest"}
		default:
			// Unknown choice: do not forbid MCP; still require reasonable reply via rubric.
		}
	case "compound_analysis_backtest":
		out.Profile = ""
		switch {
		case choiceIndicatesAnalysisOnly(choice):
			out.Profile = "stock_analysis.symbol_resolve"
			out.ForbidTools = []string{"run_strategy_backtest"}
		case strings.Contains(choice, "回测"):
			out.ForbidTools = nil
		default:
			out.ForbidTools = append([]string(nil), spec.ForbidTools...)
		}
	}
	return out
}
