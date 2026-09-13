package sessiontask

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// State is structured task context derived from prior tool calls (not token rules).
type State struct {
	Task       string
	Symbol     string
	Strategy   string
	Frequency  string
	MonthsBack string
	LastTools  []string
}

// BuildState scans session messages for the most recent probe/backtest/search context.
func BuildState(session *runtime.Session, lastTurnDomain string) State {
	if session == nil {
		return State{}
	}
	st := State{Task: strings.TrimSpace(lastTurnDomain)}
	var lastTools []string
	for i := len(session.Messages) - 1; i >= 0; i-- {
		m := session.Messages[i]
		if m.Role != llm.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for j := len(m.ToolCalls) - 1; j >= 0; j-- {
			call := m.ToolCalls[j]
			name := strings.TrimSpace(call.Name)
			if name == "" {
				continue
			}
			lastTools = prependUnique(lastTools, name)
			args := call.Arguments
			switch name {
			case "probe_bot_signal_series", "probe_bot_signal":
				st.Task = "signal_probe"
				st.Symbol = firstString(args, "code", "symbol")
				st.Strategy = firstString(args, "strategy_label", "strategy", "signal_id")
				st.Frequency = firstString(args, "frequency")
				st.MonthsBack = firstString(args, "months_back")
			case "run_strategy_backtest":
				st.Task = "backtest_run"
				st.Symbol = firstString(args, "code", "symbol")
				st.Strategy = firstString(args, "strategy_label", "strategy")
				st.Frequency = firstString(args, "frequency")
				st.MonthsBack = firstString(args, "months_back")
			case "search_code":
				if st.Symbol == "" {
					st.Symbol = firstString(args, "query", "code", "name")
				}
			}
		}
		if st.Task != "" && st.Symbol != "" && len(lastTools) >= 2 {
			break
		}
	}
	st.LastTools = lastTools
	return st
}

// RenderMarkdown returns a user-context block for fragment injection.
func (s State) RenderMarkdown() string {
	if s.Task == "" && s.Symbol == "" && len(s.LastTools) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## 当前会话任务（来自工具结果，请据此理解续轮）\n")
	if s.Task != "" {
		fmt.Fprintf(&b, "- task: %s\n", s.Task)
	}
	if s.Symbol != "" {
		fmt.Fprintf(&b, "- symbol: %s\n", s.Symbol)
	}
	if s.Strategy != "" {
		fmt.Fprintf(&b, "- strategy: %s\n", s.Strategy)
	}
	if s.Frequency != "" {
		fmt.Fprintf(&b, "- frequency: %s\n", s.Frequency)
	}
	if s.MonthsBack != "" {
		fmt.Fprintf(&b, "- months_back: %s\n", s.MonthsBack)
	}
	if len(s.LastTools) > 0 {
		fmt.Fprintf(&b, "- last_tools: %s\n", strings.Join(s.LastTools, " → "))
	}
	return strings.TrimSpace(b.String())
}

func firstString(args map[string]any, keys ...string) string {
	if len(args) == 0 {
		return ""
	}
	for _, k := range keys {
		if v, ok := args[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case fmt.Stringer:
				if s := strings.TrimSpace(t.String()); s != "" {
					return s
				}
			default:
				if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return ""
}

func prependUnique(list []string, name string) []string {
	for _, existing := range list {
		if existing == name {
			return list
		}
	}
	return append([]string{name}, list...)
}
