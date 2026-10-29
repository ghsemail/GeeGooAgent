package procedural

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const backtestRunPlaybookSkill = "strategy-backtest-run"

var legacyBacktestToolNames = map[string]struct{}{
	"generate_dca_strategy":  {},
	"loopback_strategy":      {},
	"generate_grid_strategy": {},
}

func skillNameSet(names []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out[n] = struct{}{}
		}
	}
	return out
}

func sessionHasBacktestContext(messages []llm.Message) bool {
	for _, msg := range messages {
		text := strings.ToLower(strings.TrimSpace(msg.Content))
		if text == "" {
			continue
		}
		for _, tok := range []string{
			"回测", "run_strategy_backtest", "loopback_strategy", "generate_dca_strategy",
			"smarttrade", "sar", "macd", "组合信号", "strategy-backtest",
		} {
			if strings.Contains(text, strings.ToLower(tok)) {
				return true
			}
		}
	}
	return false
}

// FilterSchemasForMatchedSkills applies skill-driven schema policy (e.g. strategy-backtest-run
// hides DCA/grid loopback tools when SmartTrade backtest is intended).
func FilterSchemasForMatchedSkills(
	schemas []llm.ToolSchema,
	matchedSkillNames []string,
	userText string,
	sessionMessages []llm.Message,
) []llm.ToolSchema {
	skills := skillNameSet(matchedSkillNames)
	if _, ok := skills[backtestRunPlaybookSkill]; !ok {
		return schemas
	}
	if !ShouldBlockLegacyBacktestTools(userText) {
		return schemas
	}
	if BacktestContinueIntent(userText) && !sessionHasBacktestContext(sessionMessages) {
		return schemas
	}
	out := make([]llm.ToolSchema, 0, len(schemas))
	for _, s := range schemas {
		if _, block := legacyBacktestToolNames[s.Name]; block {
			continue
		}
		out = append(out, s)
	}
	return out
}
