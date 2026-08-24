package tools_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestDefaultChatIncludesAllToolsetsExceptRecall(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	set := map[string]struct{}{}
	for _, name := range names {
		set[name] = struct{}{}
	}
	if _, ok := set["recall"]; ok {
		t.Fatal("recall should stay excluded from default chat allowlist")
	}
	for _, name := range []string{
		"get_single_prompt_template", "add_single_prompt_template",
		"get_custom_signal_for_skill", "add_custom_signal",
		"create_stock_premarket_report", "get_report_bot_codes",
	} {
		if _, ok := set[name]; !ok {
			t.Fatalf("%s should be in default chat (all toolsets enabled)", name)
		}
	}
	if _, ok := set["search_knowledge"]; ok {
		t.Fatal("search_knowledge must stay out of default chat allowlist")
	}
	if len(names) != 120 {
		t.Fatalf("default chat allowlist want 120, got %d", len(names))
	}
}

func TestChatToolNamesForLegacyMarketAlias(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets([]string{"market"})
	set := map[string]struct{}{}
	for _, n := range names {
		set[n] = struct{}{}
	}
	if _, ok := set["search_code"]; !ok {
		t.Fatal("legacy market alias should include search_code")
	}
	if _, ok := set["get_mcp_analysis"]; !ok {
		t.Fatal("legacy market alias should include get_mcp_analysis via analyst_runtime")
	}
	if _, ok := set["list_dca_bots"]; ok {
		t.Fatal("bot tools should not appear in market alias allowlist")
	}
}

func TestNormalizeToolsetIDs(t *testing.T) {
	t.Parallel()
	ids, err := tools.NormalizeToolsetIDs([]string{"Market", "bot_manager"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"market", "analyst_runtime", "trading_bot", "hedge_bot"}
	if len(ids) != len(want) {
		t.Fatalf("got %#v", ids)
	}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("got %#v want %#v", ids, want)
		}
	}
	if _, err := tools.NormalizeToolsetIDs([]string{"nope"}); err == nil {
		t.Fatal("expected unknown toolset error")
	}
}

func TestNormalizeToolsetIDsLegacyPromptTemplate(t *testing.T) {
	t.Parallel()
	ids, err := tools.NormalizeToolsetIDs([]string{"prompt_template"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"analyst_runtime", "prompt_admin", "custom_signal"}
	if len(ids) != len(want) {
		t.Fatalf("got %#v", ids)
	}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("got %#v want %#v", ids, want)
		}
	}
}

func TestNormalizeToolsetIDsTradingBotOnly(t *testing.T) {
	t.Parallel()
	ids, err := tools.NormalizeToolsetIDs([]string{"trading_bot"})
	if err != nil {
		t.Fatal(err)
	}
	names := tools.ChatToolNamesForToolsets(ids)
	for _, name := range names {
		if strings.Contains(name, "hdg") {
			t.Fatalf("hedge tool %s should not appear in trading_bot only", name)
		}
	}
}

func TestFormatToolsetsListingMarksActive(t *testing.T) {
	t.Parallel()
	text := tools.FormatToolsetsListing([]string{"market"})
	if !strings.Contains(text, "* market") {
		t.Fatalf("expected market marked active:\n%s", text)
	}
}

func TestAllRegisteredToolsBelongToToolset(t *testing.T) {
	t.Parallel()
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{
		AllowedHosts: []string{"127.0.0.1"},
	})
	r := tools.NewRegistry()
	tools.RegisterAll(r, tools.Deps{HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir()})

	union := map[string]struct{}{}
	for _, ts := range tools.AllToolsets() {
		for _, name := range ts.Names() {
			union[name] = struct{}{}
		}
	}
	var missing []string
	for _, name := range r.Names() {
		if _, ok := union[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("registered tools missing from toolsets: %v", missing)
	}
}

func TestAnalystRuntimeIncludesPromptIndexRead(t *testing.T) {
	t.Parallel()
	ts, ok := tools.ToolsetByID("analyst_runtime")
	if !ok {
		t.Fatal("missing analyst_runtime")
	}
	if !ts.Contains("get_single_prompt_template_by_index") {
		t.Fatal("by_index read should live in analyst_runtime with list read")
	}
	if ts.Contains("add_single_prompt_template") {
		t.Fatal("writes should not be in analyst_runtime")
	}
}

func TestReportWorkflowToolsetIncludesPostMarketIdempotency(t *testing.T) {
	t.Parallel()
	ts, ok := tools.ToolsetByID("report_workflow")
	if !ok {
		t.Fatal("missing report_workflow toolset")
	}
	if !ts.Contains("list_today_stock_postmarket_reports") {
		t.Fatal("report_workflow should include list_today_stock_postmarket_reports")
	}
	if !ts.ChatDefault {
		t.Fatal("report_workflow should be default chat")
	}
}

func TestToolsetCountsMatchDocumentation(t *testing.T) {
	t.Parallel()
	want := map[string]int{
		"market": 9, "analyst_runtime": 6, "prompt_admin": 10, "custom_signal": 7,
		"strategy": 21, "trading_bot": 15, "hedge_bot": 5, "reminder_manager": 15,
		"report_query": 7, "report_write": 8, "report_workflow": 9, "agent_meta": 9,
		"knowledge": 1,
	}
	union := map[string]struct{}{}
	for _, ts := range tools.AllToolsets() {
		if got := len(ts.Names()); got != want[ts.ID] {
			t.Fatalf("toolset %s: want %d tools, got %d", ts.ID, want[ts.ID], got)
		}
		if ts.ID == "knowledge" {
			if ts.ChatDefault {
				t.Fatal("knowledge toolset must be ChatDefault=false")
			}
		} else if !ts.ChatDefault {
			t.Fatalf("toolset %s should be ChatDefault", ts.ID)
		}
		for _, name := range ts.Names() {
			union[name] = struct{}{}
		}
	}
	if len(union) != 122 {
		t.Fatalf("toolset union want 122, got %d", len(union))
	}
}

func TestStrategyToolsExcludeAnalysisRuntime(t *testing.T) {
	t.Parallel()
	ts, ok := tools.ToolsetByID("strategy")
	if !ok {
		t.Fatal("missing strategy toolset")
	}
	for _, name := range []string{"get_mcp_analysis", "get_single_prompt_template"} {
		if ts.Contains(name) {
			t.Fatalf("%s should not be in strategy toolset", name)
		}
	}
}
