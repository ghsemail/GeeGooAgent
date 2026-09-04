package slots

import "testing"

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
