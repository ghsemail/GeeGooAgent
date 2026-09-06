package playbookexec

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestBuildProbePlanUsesLLMWhenHeuristicStockInvalid(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: `{"stock_query":"中际旭创","signal_query":"SAR MACD","signal_kind":"combination","months_back":3}`,
		}},
	}
	gw := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gw.SetSleep(func(time.Duration) {})

	session := runtime.NewSession()
	router := &Router{Gateway: gw}
	plan, note, err := router.buildProbePlan(context.Background(), Input{
		Session:  session,
		UserText: "就用SAR加MACD组合，帮我测一下中际旭创有没有买卖点",
	}, 1)
	if err != nil {
		t.Fatalf("buildProbePlan: %v", err)
	}
	if plan.StockQuery != "中际旭创" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.SignalQuery == "" {
		t.Fatal("expected signal_query")
	}
	if note == "" || note == "playbookexec plan(heuristic)" {
		t.Fatalf("expected llm plan note, got %q", note)
	}
}

func TestBuildProbePlanHeuristicFastPath(t *testing.T) {
	router := &Router{}
	plan, note, err := router.buildProbePlan(context.Background(), Input{
		UserText: "测一下中际旭创 SAR+MACD 买卖点",
	}, 1)
	if err != nil {
		t.Fatalf("buildProbePlan: %v", err)
	}
	if plan.StockQuery != "中际旭创" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if note == "" {
		t.Fatal("expected plan note")
	}
}
