package tools_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestDefaultChatToolsetsExcludeWorkflow(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	set := map[string]struct{}{}
	for _, name := range names {
		set[name] = struct{}{}
	}
	for _, name := range []string{
		"create_pre_market_report", "write_execution_log", "list_today_post_market_reports",
		"add_single_prompt_template", "add_custom_signal",
	} {
		if _, ok := set[name]; ok {
			t.Fatalf("workflow/admin tool %s should not be in default chat allowlist", name)
		}
	}
	if _, ok := set["get_bot_yesterday_attitude"]; !ok {
		t.Fatal("get_bot_yesterday_attitude should be in default chat (report_query toolset)")
	}
	if _, ok := set["get_index_signals"]; !ok {
		t.Fatal("get_index_signals should be in default chat (strategy toolset)")
	}
	if _, ok := set["get_custom_signal_for_skill"]; !ok {
		t.Fatal("get_custom_signal_for_skill should be in default chat (custom_signal reads)")
	}
	if len(names) < 20 {
		t.Fatalf("expected a substantial chat allowlist, got %d", len(names))
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

func TestAnalystRuntimeInDefaultChat(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	want := []string{"get_single_prompt_template", "get_mcp_analysis"}
	for _, name := range want {
		found := false
		for _, n := range names {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s should be in default chat via analyst_runtime toolset", name)
		}
	}
}

func TestReportWorkflowToolsetIncludesPostMarketIdempotency(t *testing.T) {
	t.Parallel()
	ts, ok := tools.ToolsetByID("report_workflow")
	if !ok {
		t.Fatal("missing report_workflow toolset")
	}
	if !ts.Contains("list_today_post_market_reports") {
		t.Fatal("report_workflow should include list_today_post_market_reports")
	}
}

func TestToolsetCountsMatchDocumentation(t *testing.T) {
	t.Parallel()
	want := map[string]int{
		"market": 9, "analyst_runtime": 4, "prompt_admin": 11, "custom_signal": 7,
		"strategy": 5, "trading_bot": 15, "hedge_bot": 5, "reminder_manager": 15,
		"report_query": 7, "report_write": 8, "report_workflow": 7, "agent_meta": 8,
	}
	union := map[string]struct{}{}
	for _, ts := range tools.AllToolsets() {
		if got := len(ts.Names()); got != want[ts.ID] {
			t.Fatalf("toolset %s: want %d tools, got %d", ts.ID, want[ts.ID], got)
		}
		for _, name := range ts.Names() {
			union[name] = struct{}{}
		}
	}
	if len(union) != 101 {
		t.Fatalf("toolset union want 101, got %d", len(union))
	}
	defaultChat := tools.ChatToolNamesForToolsets(nil)
	if len(defaultChat) != 79 {
		t.Fatalf("default chat allowlist want 79, got %d", len(defaultChat))
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

func TestPromptAdminNotInDefaultChat(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	for _, name := range names {
		if name == "edit_prompt_template" || name == "add_single_prompt_template" {
			t.Fatalf("prompt_admin tool %s should not be in default chat", name)
		}
	}
}
