package sessiontask_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/sessiontask"
)

func TestBuildStateFromProbeToolCall(t *testing.T) {
	s := runtime.NewSession()
	s.Messages = append(s.Messages,
		llm.Message{Role: llm.RoleUser, Content: "测一下 NVDA"},
		llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{
			Name: "probe_bot_signal_series",
			Arguments: map[string]any{
				"code":            "NVDA",
				"strategy_label":  "SAR+MACD",
				"frequency":       "60m",
				"months_back":     3,
			},
		}}},
	)
	st := sessiontask.BuildState(s, "signal_probe")
	if st.Symbol != "NVDA" || st.Strategy != "SAR+MACD" || st.Frequency != "60m" {
		t.Fatalf("state=%+v", st)
	}
	md := st.RenderMarkdown()
	if !strings.Contains(md, "signal_probe") || !strings.Contains(md, "NVDA") {
		t.Fatalf("markdown=%q", md)
	}
}

func TestBuildStateEmptySession(t *testing.T) {
	st := sessiontask.BuildState(runtime.NewSession(), "")
	if st.RenderMarkdown() != "" {
		t.Fatalf("expected empty markdown")
	}
}
