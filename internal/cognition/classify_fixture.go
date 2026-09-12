package cognition

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// ClassifyFixtureProvider returns canned classify JSON for tests and plan-only eval.
type ClassifyFixtureProvider struct {
	ByMessage map[string]string
	Default   string
}

// DefaultLoopClassifyFixture is the classify stub wired into cognition.Defaults()
// so agent loop tests do not require a live LLM for TurnPlan.
func DefaultLoopClassifyFixture() *ClassifyFixtureProvider {
	return &ClassifyFixtureProvider{
		ByMessage: map[string]string{
			"查一下腾讯":              FormatClassifyJSON("stock_analysis", "gather", "analyze", "loop test"),
			"腾讯多少钱":              FormatClassifyJSON("stock_analysis", "gather", "quote_price", "loop test"),
			"腾讯价格":               FormatClassifyJSON("stock_analysis", "gather", "quote_price", "loop test"),
			"腾讯现在股价多少":           FormatClassifyJSON("stock_analysis", "gather", "quote_price", "loop test"),
			"帮我分析下腾讯早上为啥下跌":      FormatClassifyJSON("stock_analysis", "gather", "technical_analysis", "loop test"),
			"创建 bot":             FormatClassifyJSON("bot_manage", "execute", "", "loop test"),
			"创建":                 FormatClassifyJSON("bot_manage", "execute", "", "loop test"),
			"删除":                 FormatClassifyJSON("bot_manage", "execute", "", "loop test"),
			"之前查过什么":             FormatClassifyJSON("chat", "talk", "", "loop test"),
			"帮我回测小米 SAR+MACD":   FormatClassifyJSON("backtest_run", "execute", "", "loop test"),
		},
		Default: FormatClassifyJSON("chat", "talk", "", "loop test default"),
	}
}

// SubAgentClassifyFixture classifies delegated sub-turns as scoped stock gather work
// without consuming chat-gateway mock responses in tests.
func SubAgentClassifyFixture() *ClassifyFixtureProvider {
	return &ClassifyFixtureProvider{
		Default: FormatClassifyJSON("stock_analysis", "gather", "analyze", "subagent"),
	}
}

func (p *ClassifyFixtureProvider) Model() string { return "classify-fixture" }

func (p *ClassifyFixtureProvider) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	user := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == llm.RoleUser {
			user = extractClassifyUserLine(msgs[i].Content)
			break
		}
	}
	body := ""
	if p != nil && p.ByMessage != nil {
		body = p.ByMessage[user]
	}
	if body == "" && p != nil {
		body = p.Default
	}
	if body == "" {
		body = `{"domain":"ambiguous","mode":"clarify","confidence":0.5,"reason":"fixture miss"}`
	}
	return &llm.Response{Content: body}, nil
}

// FormatClassifyJSON builds a classify response body for fixture providers.
func FormatClassifyJSON(domain, mode, act string, reason string) string {
	payload := map[string]any{
		"domain":     domain,
		"mode":       mode,
		"confidence": 0.95,
		"reason":     reason,
	}
	if act != "" {
		payload["act"] = act
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"domain":"%s","mode":"%s","confidence":0.95,"reason":"%s"}`, domain, mode, reason)
	}
	return string(raw)
}

func extractClassifyUserLine(content string) string {
	const prefix = "User: "
	if idx := strings.LastIndex(content, prefix); idx >= 0 {
		return strings.TrimSpace(content[idx+len(prefix):])
	}
	return strings.TrimSpace(content)
}
