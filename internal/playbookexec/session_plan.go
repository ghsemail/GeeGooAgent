package playbookexec

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

var sessionBacktestContextTokens = []string{
	"回测", "run_strategy_backtest", "loopback_strategy", "generate_dca_strategy",
	"smarttrade", "sar", "macd", "组合信号", "strategy-backtest",
}

func sessionHasBacktestContext(session *runtime.Session) bool {
	if session == nil {
		return false
	}
	for _, msg := range session.LLMMessages() {
		text := strings.ToLower(strings.TrimSpace(msg.Content))
		if text == "" {
			continue
		}
		for _, tok := range sessionBacktestContextTokens {
			if strings.Contains(text, strings.ToLower(tok)) {
				return true
			}
		}
	}
	return false
}

func enrichPlanFromSession(plan *BacktestRunPlan, session *runtime.Session) {
	if plan == nil || session == nil {
		return
	}
	if strings.TrimSpace(plan.SignalQuery) == "" {
		if label := lastStrategyLabel(session); label != "" {
			plan.SignalQuery = label
			plan.SignalKind = "combination"
		}
	}
	if strings.TrimSpace(plan.StockQuery) == "" {
		if q := lastConfirmedStock(session); q != "" {
			plan.StockQuery = q
		}
	}
	msgs := session.LLMMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != llm.RoleUser {
			continue
		}
		content := strings.TrimSpace(msgs[i].Content)
		if content == "" || !isBacktestSlotMessage(content) {
			continue
		}
		if strings.TrimSpace(plan.StockQuery) == "" {
			if q := extractStockQuery(content); q != "" {
				plan.StockQuery = q
			}
		}
		if strings.TrimSpace(plan.SignalQuery) == "" {
			applySignalHeuristics(plan, content)
		}
		if strings.TrimSpace(plan.StockQuery) != "" && strings.TrimSpace(plan.SignalQuery) != "" {
			break
		}
	}
	normalizeBacktestPlan(plan)
}

func isBacktestSlotMessage(content string) bool {
	return strings.Contains(content, "回测") || procedural.BacktestContinueIntent(content)
}

func lastConfirmedStock(session *runtime.Session) string {
	if session == nil {
		return ""
	}
	msgs := session.LLMMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != llm.RoleAssistant {
			continue
		}
		content := msgs[i].Content
		if !strings.Contains(content, "回测") {
			continue
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "##") {
				if q := extractStockQuery(line); q != "" {
					return q
				}
			}
		}
	}
	return ""
}

func lastStrategyLabel(session *runtime.Session) string {
	if session == nil {
		return ""
	}
	msgs := session.LLMMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != llm.RoleAssistant {
			continue
		}
		content := msgs[i].Content
		if !strings.Contains(content, "回测") && !strings.Contains(content, "收益率") {
			continue
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "- ")
			if strings.HasPrefix(line, "**策略**：") {
				return strings.TrimSpace(strings.TrimPrefix(line, "**策略**："))
			}
			if strings.HasPrefix(line, "**策略**:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "**策略**:"))
			}
		}
	}
	return ""
}

func applySignalHeuristics(plan *BacktestRunPlan, content string) {
	upper := strings.ToUpper(content)
	switch {
	case strings.Contains(upper, "SAR") && strings.Contains(upper, "MACD"):
		plan.SignalKind = "combination"
		plan.SignalQuery = "SAR MACD"
	case strings.Contains(upper, "RSI"):
		plan.SignalKind = "indicator"
		plan.SignalQuery = "RSI"
	case strings.Contains(upper, "SAR"):
		plan.SignalKind = "indicator"
		plan.SignalQuery = "SAR"
	case strings.Contains(content, "共振"):
		plan.SignalKind = "combination"
		if strings.TrimSpace(plan.SignalQuery) == "" {
			plan.SignalQuery = "共振"
		}
	case strings.Contains(content, "组合"):
		plan.SignalKind = "combination"
		if strings.TrimSpace(plan.SignalQuery) == "" {
			plan.SignalQuery = "组合"
		}
	}
}

func recentSessionContext(session *runtime.Session, maxMessages int) string {
	if session == nil || maxMessages <= 0 {
		return ""
	}
	msgs := session.LLMMessages()
	if len(msgs) == 0 {
		return ""
	}
	start := 0
	if len(msgs) > maxMessages {
		start = len(msgs) - maxMessages
	}
	var b strings.Builder
	for _, msg := range msgs[start:] {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		b.WriteString(string(msg.Role))
		b.WriteString(": ")
		b.WriteString(truncate(content, 400))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// FilterLegacyBacktestTools removes DCA/grid loopback tools when SmartTrade backtest is intended.
func FilterLegacyBacktestTools(schemas []llm.ToolSchema, userText string, session *runtime.Session) []llm.ToolSchema {
	if !procedural.ShouldBlockLegacyBacktestTools(userText) {
		return schemas
	}
	if procedural.BacktestContinueIntent(userText) && !sessionHasBacktestContext(session) {
		return schemas
	}
	blocked := map[string]struct{}{
		"generate_dca_strategy":  {},
		"loopback_strategy":      {},
		"generate_grid_strategy": {},
	}
	out := make([]llm.ToolSchema, 0, len(schemas))
	for _, s := range schemas {
		if _, ok := blocked[s.Name]; ok {
			continue
		}
		out = append(out, s)
	}
	return out
}
