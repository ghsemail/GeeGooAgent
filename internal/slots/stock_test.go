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

func TestExtractStockQueryProbeWithMaiMaiDian(t *testing.T) {
	cases := []struct {
		msg  string
		want string
	}{
		{
			msg:  "就用SAR加MACD组合，帮我测一下中际旭创有没有买卖点",
			want: "中际旭创",
		},
		{
			msg:  "帮我看看中际旭创有没有买卖点",
			want: "中际旭创",
		},
		{
			msg:  "测一下中际旭创有没有买点",
			want: "中际旭创",
		},
		{
			msg:  "测一下中际旭创有没有卖点",
			want: "中际旭创",
		},
	}
	for _, tc := range cases {
		if got := ExtractStockQuery(tc.msg); got != tc.want {
			t.Fatalf("ExtractStockQuery(%q)=%q want %q", tc.msg, got, tc.want)
		}
	}
}

func TestExtractStockQueryRejectsMaiMaiDianFragments(t *testing.T) {
	for _, msg := range []string{"有没有买卖点", "卖点", "买点", "有没有买"} {
		if got := ExtractStockQuery(msg); got != "" {
			t.Fatalf("ExtractStockQuery(%q)=%q want empty", msg, got)
		}
	}
}

func TestNormalizeStockCandidate(t *testing.T) {
	cases := map[string]string{
		"中际旭创有没有买":   "中际旭创",
		"中际旭创有没有买卖点": "中际旭创",
		"腾讯有没有卖点":    "腾讯",
	}
	for in, want := range cases {
		if got := normalizeStockCandidate(in); got != want {
			t.Fatalf("normalizeStockCandidate(%q)=%q want %q", in, got, want)
		}
	}
	for _, bad := range []string{"卖点", "买点", "有没有买"} {
		if _, ok := acceptStockCandidate(bad); ok {
			t.Fatalf("acceptStockCandidate(%q) should reject", bad)
		}
	}
}

func TestExtractExplicitStockReferenceIgnoresKLineFollowUp(t *testing.T) {
	msg := "可以，分析下技术面的价格和K线图"
	if got := ExtractExplicitStockReference(msg); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestExtractExplicitStockReferenceFindsSwitch(t *testing.T) {
	if got := ExtractExplicitStockReference("那就换成贵州茅台吧"); got != "贵州茅台" && got != "茅台" {
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
