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
}
