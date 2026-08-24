package catalog_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

func TestNeedsMCPTokenDefaults(t *testing.T) {
	t.Parallel()
	if catalog.NeedsMCPToken("list_smart_trades") != true {
		t.Fatal("list_smart_trades should require mcp_token")
	}
	if catalog.NeedsMCPToken("get_position") != true {
		t.Fatal("get_position should require mcp_token")
	}
	for _, name := range []string{"search_code", "get_index_signals", "get_signal_combinations"} {
		if catalog.NeedsMCPToken(name) {
			t.Fatalf("%s should not require mcp_token", name)
		}
	}
	for _, name := range []string{"add_index_signal", "get_custom_signal_for_skill", "add_custom_signal"} {
		if !catalog.NeedsMCPToken(name) {
			t.Fatalf("%s should require mcp_token", name)
		}
	}
}

func TestUsesBotMCPProxy(t *testing.T) {
	t.Parallel()
	if !catalog.UsesBotMCPProxy("get_custom_signal_for_skill") {
		t.Fatal("custom skill read should use mcp-api proxy")
	}
	if catalog.UsesBotMCPProxy("get_custom_strategy_definitions") {
		t.Fatal("strategy definitions should stay on catalog-api")
	}
	if catalog.UsesSignalCatalog("get_custom_signal_for_skill") {
		t.Fatal("proxied tools should not use direct catalog-api")
	}
	if !catalog.UsesSignalCatalog("get_custom_strategy_definitions") {
		t.Fatal("definitions should use direct catalog-api")
	}
}
