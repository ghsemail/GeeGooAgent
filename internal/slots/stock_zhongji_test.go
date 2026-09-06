package slots

import "testing"

func TestExtractStockQueryAnalysisZhongji(t *testing.T) {
	msg := "帮我分析一下中际旭创"
	got := ExtractStockQuery(msg)
	if got != "中际旭创" {
		t.Fatalf("ExtractStockQuery(%q)=%q want 中际旭创", msg, got)
	}
	if explicit := ExtractExplicitStockReference(msg); explicit != "中际旭创" {
		t.Fatalf("ExtractExplicitStockReference(%q)=%q", msg, explicit)
	}
}

func TestExtractStockQueryAnalysisXiaomi(t *testing.T) {
	if got := ExtractStockQuery("帮我分析一下小米"); got != "小米" {
		t.Fatalf("got %q", got)
	}
}
