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
	"github.com/ghsemail/GeeGooAgent/internal/playbookexec"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func analysisPlaybookTool(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
	switch req.Name {
	case "search_code":
		return tools.Result{
			Status: tools.StatusOK,
			Data: map[string]any{
				"items": []any{map[string]any{"code": "00700.HK", "name": "腾讯控股", "market": "HK"}},
			},
		}
	case "get_single_prompt_template":
		return tools.Result{
			Status: tools.StatusOK,
			Data: map[string]any{"selected_prompt_id": "p1", "selected_prompt_name": "股价分析"},
		}
	case "get_mcp_analysis":
		return tools.Result{
			Status:  tools.StatusOK,
			Summary: "mock analysis",
			Data:    map[string]any{"analysis_result": "腾讯技术面偏强。"},
		}
	default:
		return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
	}
}

func analysisPlaybookRouter() *playbookexec.Router {
	return &playbookexec.Router{RunTool: analysisPlaybookTool}
}

func TestTurnPlanDoesNotRunBacktestOnAnalysis(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: "腾讯技术面偏强。",
		}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))

	var backtestCalls atomic.Int32
	loop.SetPlaybookRouter(&playbookexec.Router{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			if req.Name == "run_strategy_backtest" {
				backtestCalls.Add(1)
			}
			return analysisPlaybookTool(ctx, req, toolCtx)
		},
	})

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
	if backtestCalls.Load() != 0 {
		t.Fatal("analysis turn must not run backtest playbook")
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
	loop.SetPlaybookRouter(analysisPlaybookRouter())

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

func TestTurnPlanRunsPlaybookOnlyForExplicitBacktest(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "unused"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))

	var ran atomic.Int32
	loop.SetPlaybookRouter(&playbookexec.Router{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			ran.Add(1)
			return tools.Result{Status: tools.StatusError, Summary: "stop"}
		},
	})

	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "帮我回测小米 SAR+MACD", tools.Context{}, nil)
	if ran.Load() == 0 {
		t.Fatal("expected playbook tools for explicit backtest")
	}
}

func TestStockAnalysisEmitsGateBeforeTools(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "腾讯技术面偏强。"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetPlaybookRouter(analysisPlaybookRouter())

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

	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "unused"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetSkillLoader(procedural.NewLoader(dir), 4)
	loop.SetPlaybookRouter(&playbookexec.Router{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			return tools.Result{Status: tools.StatusError, Summary: "stop"}
		},
	})

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
	loop.SetRetrievalGate(gateMock, nil, 4)

	var statuses []string
	loop.SetProgress(func(event string, data map[string]any) {
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
	foundSkip := false
	for _, msg := range statuses {
		if strings.Contains(msg, "澄清轮次，跳过记忆检索") {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Fatalf("expected clarify gate skip in statuses=%v", statuses)
	}
	if clarifyQuestion != "你是想做哪一件？" {
		t.Fatalf("question=%q", clarifyQuestion)
	}
}
