package postmarket_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/postmarket"
)

func TestBuildPostMarketSummaryOneLiner(t *testing.T) {
	ws := memory.StockWorkspace{
		Code: "00700.HK", StockName: "腾讯控股", ChangePct: 1.08, SessionBias: "bullish",
	}
	out := postmarket.BuildPostMarketSummaryOneLiner(ws, "bullish", "aligned")
	if out == "" {
		t.Fatal("empty")
	}
	if len([]rune(out)) > 200 {
		t.Fatalf("too long: %d", len([]rune(out)))
	}
	for _, bad := range []string{"今日行情", "小时级", "…"} {
		if strings.Contains(out, bad) {
			t.Fatalf("unexpected %q in %q", bad, out)
		}
	}
}
