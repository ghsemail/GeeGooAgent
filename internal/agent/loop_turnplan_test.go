package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestTurnPlanDoesNotRunBacktestOnAnalysis(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "run_strategy_backtest",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			t.Fatal("analysis turn must not call backtest tool")
			return tools.Result{}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: "腾讯技术面偏强。",
		}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	withClassifyPlanner(loop, classifyFixture(nil))

	var domain string
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "turn_plan" {
			if d, ok := data["domain"].(string); ok {
				domain = d
			}
		}
	})

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "腾讯现在怎么样", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if domain != string(cognition.DomainStockAnalysis) {
		t.Fatalf("domain=%q", domain)
	}
	if !strings.Contains(result.AssistantText, "腾讯") {
		t.Fatalf("expected analysis reply, got %q", result.AssistantText)
	}
}

func TestTurnPlanFollowsLastDomainOnSymbolSwitch(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "腾讯技术面偏强。"},
			{Content: "茅台继续观察。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	withClassifyPlanner(loop, classifyFixture(nil))

	var domains []string
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "turn_plan" {
			if d, ok := data["domain"].(string); ok {
				domains = append(domains, d)
			}
		}
	})

	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "腾讯现在怎么样", tools.Context{}, nil)
	_ = loop.RunTurn(context.Background(), session, "换成贵州茅台", tools.Context{}, nil)
	if len(domains) != 2 {
		t.Fatalf("domains=%v", domains)
	}
	if domains[0] != string(cognition.DomainStockAnalysis) || domains[1] != string(cognition.DomainStockAnalysis) {
		t.Fatalf("expected sticky analysis, got %v", domains)
	}
	if session.LastTurnDomain != string(cognition.DomainStockAnalysis) {
		t.Fatalf("LastTurnDomain=%q", session.LastTurnDomain)
	}
}

func TestTurnPlanExecutesBacktestViaReAct(t *testing.T) {
	registry := tools.NewRegistry()
	var backtestCalls atomic.Int32
	registry.Register(tools.Tool{
		Name: "run_strategy_backtest",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			backtestCalls.Add(1)
			return tools.Result{Status: tools.StatusOK, Summary: "mock backtest done"}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "b1", Name: "run_strategy_backtest", Arguments: map[string]any{"code": "1810.HK"}}}},
			{Content: "小米 SAR+MACD 回测已完成。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	withClassifyPlanner(loop, classifyFixture(nil))

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "帮我回测小米 SAR+MACD", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if backtestCalls.Load() == 0 {
		t.Fatal("expected main LLM to call run_strategy_backtest via ReAct")
	}
}

func TestStockAnalysisEmitsGateBeforeTools(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "search_code",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{
				Status: tools.StatusOK,
				Data:   map[string]any{"items": []any{map[string]any{"code": "00700.HK", "name": "腾讯控股"}}},
			}
		},
	})
	registry.Register(tools.Tool{
		Name: "get_mcp_analysis",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"analysis_result": "腾讯现价 380 港元。"}}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "s1", Name: "search_code", Arguments: map[string]any{"regex": "腾讯"}}}},
			{ToolCalls: []llm.ToolCall{{ID: "a1", Name: "get_mcp_analysis", Arguments: map[string]any{"code": "00700.HK"}}}},
			{Content: "腾讯现价约 380 港元，技术面偏强。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	withClassifyPlanner(loop, classifyFixture(nil))

	var events []string
	loop.SetProgress(func(event string, data map[string]any) {
		events = append(events, event)
	})

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "帮我查一下腾讯的股价", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}

	gateAt, toolAt := -1, -1
	for i, ev := range events {
		if ev == "gate" && gateAt < 0 {
			gateAt = i
		}
		if ev == "tool_start" && toolAt < 0 {
			toolAt = i
		}
	}
	if gateAt < 0 {
		t.Fatalf("missing gate event in %v", events)
	}
	if toolAt < 0 {
		t.Fatalf("missing tool_start in %v", events)
	}
	if gateAt > toolAt {
		t.Fatalf("gate after tool_start: gate=%d tool=%d events=%v", gateAt, toolAt, events)
	}
}

func TestToolFirstSkipEmitsGateEvent(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "strategy-backtest-run")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: strategy-backtest-run\ndescription: 回测、跑回测、SAR MACD\n---\n\n先解析标的再跑回测。\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "run_strategy_backtest",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "mock backtest"}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "b1", Name: "run_strategy_backtest", Arguments: map[string]any{}}}},
			{Content: "回测完成。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	withClassifyPlanner(loop, classifyFixture(nil))
	loop.SetSkillLoader(procedural.NewLoader(dir), 4)

	var gateDecision string
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "gate" {
			if d, ok := data["decision"].(string); ok {
				gateDecision = d
			}
			if r, ok := data["reason"].(string); ok && strings.Contains(r, "tool-first") {
				gateDecision = "skip:" + r
			}
		}
	})

	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "帮我回测小米 SAR+MACD", tools.Context{}, nil)
	if !strings.HasPrefix(gateDecision, "skip") {
		t.Fatalf("expected skip gate, got %q", gateDecision)
	}
}

func TestAmbiguousTurnSkipsGateAndUsesPresetClarify(t *testing.T) {
	gateMock := &llm.MockProvider{
		Responses: []*llm.Response{{Content: `{"retrieve":false,"query":"","reason":"test"}`}},
	}
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "好的，继续分析。"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	withClassifyPlanner(loop, classifyFixture(nil))
	loop.SetRetrievalGate(gateMock, nil, 4)

	var statuses []string
	var events []string
	loop.SetProgress(func(event string, data map[string]any) {
		events = append(events, event)
		if event != "status" {
			return
		}
		if msg, ok := data["message"].(string); ok {
			statuses = append(statuses, msg)
		}
	})

	var clarifyQuestion string
	toolCtx := tools.Context{
		Interactive: true,
		ClarifyFn: func(_ context.Context, question string, choices []string) (string, bool) {
			clarifyQuestion = question
			if len(choices) != 4 {
				t.Fatalf("choices=%v", choices)
			}
			return "个股/指标分析", true
		},
	}

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "MACD", toolCtx, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	turnStarts := 0
	sawClarify := false
	for _, ev := range events {
		if ev == "turn_start" {
			turnStarts++
		}
		if ev == "clarify" {
			sawClarify = true
		}
	}
	if turnStarts != 1 {
		t.Fatalf("nested RunTurn would emit extra turn_start, got %d events=%v", turnStarts, events)
	}
	if !sawClarify {
		t.Fatalf("missing clarify event in %v", events)
	}
	for _, msg := range statuses {
		if strings.Contains(msg, "辅助模型推理中") || strings.Contains(msg, "正在调用辅助模型") {
			t.Fatalf("clarify turn must not wait on auxiliary model: %q statuses=%v", msg, statuses)
		}
	}
	if clarifyQuestion != "你是想做哪一件？" {
		t.Fatalf("question=%q", clarifyQuestion)
	}
	if session.LastTurnDomain != string(cognition.DomainStockAnalysis) {
		t.Fatalf("after choice LastTurnDomain=%q", session.LastTurnDomain)
	}
}
