package domaincatalog

import "testing"

func TestToolsAllowFromToolsets(t *testing.T) {
	tools := ToolsAllow(DomainStockAnalysis)
	if !contains(tools, "search_code") {
		t.Fatalf("stock analysis missing search_code: %v", tools)
	}
	if !contains(tools, "get_mcp_analysis") {
		t.Fatalf("stock analysis missing get_mcp_analysis: %v", tools)
	}
	if contains(tools, "run_strategy_backtest") {
		t.Fatal("stock analysis should not include backtest tool")
	}
}

func TestBotManageIncludesTradingBots(t *testing.T) {
	tools := ToolsAllow(DomainBotManage)
	for _, want := range []string{"list_dca_bots", "list_grid_bots", "list_smart_trades", "list_hdg_bots"} {
		if !contains(tools, want) {
			t.Fatalf("bot_manage missing %s: %v", want, tools)
		}
	}
}

func TestMakePlanAmbiguous(t *testing.T) {
	spec := MakePlan(DomainAmbiguous)
	if spec.Mode != "clarify" || len(spec.ClarifyChoices) != 4 {
		t.Fatalf("ambiguous spec=%+v", spec)
	}
}

func contains(items []string, want string) bool {
	for _, s := range items {
		if s == want {
			return true
		}
	}
	return false
}
