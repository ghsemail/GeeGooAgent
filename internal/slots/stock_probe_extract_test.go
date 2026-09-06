package slots

import "testing"

func TestExtractStockQueryProbeBareStock(t *testing.T) {
	msg := "帮我看看中际旭创有没有买卖点"
	got := ExtractStockQuery(msg)
	if got != "中际旭创" {
		t.Fatalf("ExtractStockQuery(%q)=%q want 中际旭创", msg, got)
	}
	if !StockQueryPlausible(got) {
		t.Fatalf("StockQueryPlausible(%q)=false", got)
	}
}
