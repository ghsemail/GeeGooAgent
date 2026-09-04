package agent_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/playbookexec"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestTurnPlanDoesNotRunBacktestOnAnalysis(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: "腾讯技术面偏强。",
		}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))

	var playbookCalls atomic.Int32
	loop.SetPlaybookRouter(&playbookexec.Router{
		RunTool: func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
			playbookCalls.Add(1)
			return tools.Result{Status: tools.StatusOK, Summary: "should not run"}
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
	if playbookCalls.Load() != 0 {
		t.Fatal("analysis turn must not enter backtest playbook")
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
