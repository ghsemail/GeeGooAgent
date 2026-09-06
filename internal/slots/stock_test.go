package slots

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestExtractStockQueryProbeMessage(t *testing.T) {
	got := ExtractStockQuery("测一下中际旭创 SAR+MACD 买卖点")
	if got != "中际旭创" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractStockQueryZhongji(t *testing.T) {
	if got := ExtractStockQuery("帮我回测一下中际旭创"); got != "中际旭创" {
		t.Fatalf("got %q", got)
	}
}

func TestIsLikelyStockUtterance(t *testing.T) {
	if !IsLikelyStockUtterance("中际旭创呢") {
		t.Fatal("colloquial stock name should match")
	}
	if IsLikelyStockUtterance("MACD") {
		t.Fatal("indicator alone should not match")
	}
	if IsLikelyStockUtterance("daily") {
		t.Fatal("frequency word should not match")
	}
}

func TestLooksLikeStockQueryRejectsDaily(t *testing.T) {
	if LooksLikeStockQuery("DAILY") {
		t.Fatal("DAILY must be rejected")
	}
}

func TestPickStockRowAutoResolvesTencentWithoutClarify(t *testing.T) {
	items := []map[string]any{
		{"code": "00700.HK", "name": "腾讯控股"},
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
	}
	var events []string
	ctx := tools.Context{
		Progress: func(event string, _ map[string]any) {
			events = append(events, event)
		},
		ClarifyFn: func(_ context.Context, _ string, _ []string) (string, bool) {
			t.Fatal("clarify should not run for common alias 腾讯")
			return "", false
		},
	}
	row, err := pickStockRow(context.Background(), ctx, "腾讯", items)
	if err != nil {
		t.Fatal(err)
	}
	if row["code"] != "00700.HK" {
		t.Fatalf("row=%v", row)
	}
	for _, ev := range events {
		if ev == "clarify" {
			t.Fatalf("unexpected clarify event: %v", events)
		}
	}
}
