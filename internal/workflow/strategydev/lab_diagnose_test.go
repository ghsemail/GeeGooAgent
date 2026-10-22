package strategydev

import "testing"

func TestParseLabDetailRef(t *testing.T) {
	side, idx, ok := ParseLabDetailRef("buy:3")
	if !ok || side != "buy" || idx != 3 {
		t.Fatalf("got %s %d %v", side, idx, ok)
	}
	if _, _, ok := ParseLabDetailRef("nope"); ok {
		t.Fatal("expected invalid ref")
	}
}

func TestLightKlineSummaryAtIndex(t *testing.T) {
	series := []any{
		map[string]any{"index": 0, "patterns": []any{map[string]any{"name": "hammer", "direction": "bull"}}},
	}
	sum := LightKlineSummaryAtIndex(series, 0)
	if sum == "" || sum == "无明显形态" {
		t.Fatalf("summary=%q", sum)
	}
}

func TestPriceAnalysisSummaryLine(t *testing.T) {
	pa := &PriceAnalysis{Summary: "偏多", PriceScore: 12.3}
	if PriceAnalysisSummaryLine(pa) != "偏多" {
		t.Fatal("expected summary field")
	}
}
