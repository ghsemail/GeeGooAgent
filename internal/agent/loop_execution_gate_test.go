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
	if len(out) != 2 {
		t.Fatalf("subagent profile should expose only delegate_tasks+clarify, got %d tools", len(out))
	}
	names := map[string]bool{}
	for _, s := range out {
		names[s.Name] = true
	}
	if !names["delegate_tasks"] || !names["clarify"] {
		t.Fatalf("subagent tools=%v", names)
	}
}

func TestApplyTurnToolSchemasMultiSymbolOrchestrator(t *testing.T) {
	in := []llm.ToolSchema{
		{Name: "search_code"}, {Name: "get_current_price"}, {Name: "get_mcp_analysis"},
		{Name: "delegate_tasks"}, {Name: "delegate_task"}, {Name: "clarify"},
		{Name: "fetch_market_news"},
	}
	plan := cognition.TurnPlan{
		Domain: cognition.DomainStockAnalysis,
		Mode:   cognition.ModeGather,
		Act:    domaincatalog.StockActMultiSymbol,
		ToolsAllow: []string{"delegate_tasks"},
	}
	out := applyTurnToolSchemas(in, plan)
	if len(out) != 2 {
		t.Fatalf("orchestrator turn want 2 tools, got %d: %v", len(out), out)
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
