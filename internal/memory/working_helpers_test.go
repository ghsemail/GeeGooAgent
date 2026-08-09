package memory

import (
	"strings"
	"testing"
)

func TestFormatPositionSummaryHumanReadable(t *testing.T) {
	got := formatPositionSummary(map[string]any{
		"position": 100.0, "cost_price": 614.7, "can_sell_qty": 100.0,
		"pl_val": -13590.0, "pl_ratio": -22.11,
	})
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	for _, bad := range []string{"position=", "cost_price=", "pl_val=", "pl_ratio="} {
		if strings.Contains(got, bad) {
			t.Fatalf("expected human text, got %q", got)
		}
	}
	for _, want := range []string{"持仓 100 股", "成本价 614.7", "浮动盈亏", "-22.11%"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
