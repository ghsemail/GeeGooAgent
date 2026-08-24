package agent_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunTurnInjectsClockBeforeLatestUser(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	frozen := time.Date(2026, 8, 24, 14, 50, 0, 0, loc)
	agent.SetClockNowForTest(func() time.Time { return frozen })
	t.Cleanup(func() { agent.SetClockNowForTest(nil) })

	var seen []llm.Message
	provider := &recordingChatProvider{
		onChat: func(messages []llm.Message) *llm.Response {
			seen = append([]llm.Message(nil), messages...)
			return &llm.Response{Content: "ok"}
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	session := runtime.NewSession()
	soul := session.Messages[0].Content

	result := loop.RunTurn(context.Background(), session, "帮我分析下腾讯早上为啥下跌", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if len(seen) < 2 {
		t.Fatalf("expected system+user at minimum, got %d messages", len(seen))
	}
	if seen[0].Role != llm.RoleSystem || seen[0].Content != soul {
		t.Fatalf("leading system prompt must stay stable for prefix cache")
	}
	if strings.Contains(seen[0].Content, "当前时间") {
		t.Fatal("clock leaked into stable system prompt")
	}

	lastUser := -1
	for i, m := range seen {
		if m.Role == llm.RoleUser && m.Content == "帮我分析下腾讯早上为啥下跌" {
			lastUser = i
		}
	}
	if lastUser <= 0 {
		t.Fatalf("user message missing: %+v", seen)
	}
	clockMsg := seen[lastUser-1]
	if clockMsg.Role != llm.RoleSystem {
		t.Fatalf("clock role=%s want system", clockMsg.Role)
	}
	for _, want := range []string{"当前时间：2026-08-24 14:50", "星期一", "Asia/Shanghai"} {
		if !strings.Contains(clockMsg.Content, want) {
			t.Fatalf("clock missing %q in:\n%s", want, clockMsg.Content)
		}
	}
}
