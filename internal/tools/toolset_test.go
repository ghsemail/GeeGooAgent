package tools_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestDefaultChatToolsetsExcludeWorkflow(t *testing.T) {
	t.Parallel()
	names := tools.ChatToolNamesForToolsets(nil)
	for _, name := range names {
		if name == "create_pre_market_report" || name == "write_execution_log" {
			t.Fatalf("workflow tool %s should not be in default chat allowlist", name)
		}
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
