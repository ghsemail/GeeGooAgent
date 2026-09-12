package agent

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestTryExecutionProfileRetryOnMissingMCP(t *testing.T) {
	loop := &Loop{executionProfileMaxRetries: 1}
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "分析价格走势"})
	messages := session.LLMMessages()
	turnPlan := cognition.TurnPlan{
		Domain: cognition.DomainStockAnalysis,
		Act:    domaincatalog.StockActTechnicalAnalysis,
	}
	records := []runtime.StepRecord{
		{Kind: "tool", ToolName: "search_code"},
		{Kind: "tool", ToolName: "get_current_price"},
	}
	retries := 1
	var events []string
	loop.onProgress = func(event string, _ map[string]any) { events = append(events, event) }

	if !loop.tryExecutionProfileRetry(context.Background(), session, &messages, turnPlan, records, &retries) {
		t.Fatal("expected execution profile retry")
	}
	if retries != 0 {
		t.Fatalf("retries=%d", retries)
	}
	if len(session.LLMMessages()) < 2 {
		t.Fatal("expected hint message appended")
	}
	if events[len(events)-1] != "execution_retry" {
		t.Fatalf("events=%v", events)
	}
}

func TestTryExecutionProfileRetryPassesQuoteSnapshot(t *testing.T) {
	loop := &Loop{executionProfileMaxRetries: 1}
	session := runtime.NewSession()
	turnPlan := cognition.TurnPlan{
		Domain: cognition.DomainStockAnalysis,
		Act:    domaincatalog.StockActQuotePrice,
	}
	records := []runtime.StepRecord{
		{Kind: "tool", ToolName: "search_code"},
		{Kind: "tool", ToolName: "get_current_price"},
	}
	retries := 1
	messages := session.LLMMessages()
	if loop.tryExecutionProfileRetry(context.Background(), session, &messages, turnPlan, records, &retries) {
		t.Fatal("quote snapshot should pass without retry")
	}
}

func TestFilterExecutionProfileSchemas(t *testing.T) {
	in := []llm.ToolSchema{{Name: "search_code"}, {Name: "get_current_price"}, {Name: "get_mcp_analysis"}}

	out := filterExecutionProfileSchemas(in, domaincatalog.ProfileStockPriceViaMCP)
	for _, s := range out {
		if s.Name == "get_current_price" {
			t.Fatal("get_current_price should be filtered for mcp profile")
		}
	}

	out = filterExecutionProfileSchemas(in, domaincatalog.ProfileStockPriceSnapshot)
	for _, s := range out {
		if s.Name == "get_mcp_analysis" {
			t.Fatal("get_mcp_analysis should be filtered for snapshot profile")
		}
	}

	subagentIn := append([]llm.ToolSchema{}, in...)
	subagentIn = append(subagentIn, llm.ToolSchema{Name: "delegate_tasks"}, llm.ToolSchema{Name: "clarify"})
	out = filterExecutionProfileSchemas(subagentIn, domaincatalog.ProfileSubagentMultiStock)
	if len(out) != len(subagentIn) {
		t.Fatalf("subagent profile must not hard-filter schemas, got %d want %d", len(out), len(subagentIn))
	}
}

func TestApplyTurnToolSchemasMultiSymbolKeepsStockTools(t *testing.T) {
	in := []llm.ToolSchema{
		{Name: "search_code"}, {Name: "get_current_price"}, {Name: "get_mcp_analysis"},
		{Name: "delegate_tasks"}, {Name: "delegate_task"}, {Name: "clarify"},
		{Name: "fetch_market_news"},
	}
	plan := cognition.TurnPlan{
		Domain:     cognition.DomainStockAnalysis,
		Mode:       cognition.ModeGather,
		Act:        domaincatalog.StockActMultiSymbol,
		ToolsAllow: []string{"search_code", "get_current_price", "get_mcp_analysis", "delegate_tasks"},
	}
	out := applyTurnToolSchemas(in, plan)
	names := map[string]bool{}
	for _, s := range out {
		names[s.Name] = true
	}
	for _, want := range []string{"search_code", "get_current_price", "get_mcp_analysis", "delegate_tasks", "clarify"} {
		if !names[want] {
			t.Fatalf("multi-symbol turn should expose %s, got %v", want, names)
		}
	}
}

func TestTryExecutionProfileRetryOnSubagentDirectTools(t *testing.T) {
	loop := &Loop{executionProfileMaxRetries: 1}
	session := runtime.NewSession()
	messages := session.LLMMessages()
	turnPlan := cognition.TurnPlan{
		Domain: cognition.DomainStockAnalysis,
		Act:    domaincatalog.StockActMultiSymbol,
	}
	records := []runtime.StepRecord{{Kind: "tool", ToolName: "search_code"}}
	retries := 1

	if !loop.tryExecutionProfileRetry(context.Background(), session, &messages, turnPlan, records, &retries) {
		t.Fatal("expected execution profile retry for direct search_code on subagent profile")
	}
	if retries != 0 {
		t.Fatalf("retries=%d", retries)
	}
}
