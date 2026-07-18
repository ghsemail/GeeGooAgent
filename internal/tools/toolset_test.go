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
	for _, name := range []string{"create_pre_market_report", "write_execution_log", "list_today_post_market_reports"} {
		if _, ok := set[name]; ok {
			t.Fatalf("workflow-only tool %s should not be in default chat allowlist", name)
		}
	}
	if _, ok := set["get_bot_yesterday_attitude"]; !ok {
		t.Fatal("get_bot_yesterday_attitude should be in default chat (shared with market toolset)")
	}
	if len(names) < 20 {
		t.Fatalf("expected a substantial chat allowlist, got %d", len(names))
	}
}

func TestChatToolNamesForMarketOnly(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets([]string{"market"})
	set := map[string]struct{}{}
	for _, n := range names {
		set[n] = struct{}{}
	}
	if _, ok := set["search_code"]; !ok {
		t.Fatal("market toolset should include search_code")
	}
	if _, ok := set["list_dca_bots"]; ok {
		t.Fatal("bot tools should not appear in market-only allowlist")
	}
}

func TestNormalizeToolsetIDs(t *testing.T) {
	t.Parallel()
	ids, err := tools.NormalizeToolsetIDs([]string{"Market", "bot_manager"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "market" || ids[1] != "bot_manager" {
		t.Fatalf("got %#v", ids)
	}
	if _, err := tools.NormalizeToolsetIDs([]string{"nope"}); err == nil {
		t.Fatal("expected unknown toolset error")
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

func TestPromptTemplateToolsetNotInDefaultChat(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	for _, name := range names {
		if strings.Contains(name, "prompt_template") && name != "get_single_prompt_template" {
			t.Fatalf("prompt CRUD %s should not be in default chat", name)
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
