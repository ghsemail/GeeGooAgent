package priceanalysis

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPriceAnalysisRowShape(t *testing.T) {
	bars := buildTrendBars(90, "up")
	row := BuildPriceAnalysisRow(bars, "60m", "60m")
	if row.Timeframe != "60m" {
		t.Fatalf("timeframe=%s", row.Timeframe)
	}
	if row.Conclusion == "" {
		t.Fatal("missing conclusion")
	}
	if !strings.Contains(row.Structure, "高 ") || !strings.Contains(row.Structure, "低 ") {
		t.Fatalf("structure=%q", row.Structure)
	}
	if row.BosChoch == "" {
		t.Fatal("missing bos/choch")
	}
	if row.VolumeNote == "" || row.VolumeNote == "-" {
		t.Fatalf("volume note=%q", row.VolumeNote)
	}
}

func TestBuildPriceAnalysisFromBarsPayload(t *testing.T) {
	bars := map[string][]any{
		"5m":     buildTrendBars(90, "up"),
		"60m":    buildTrendBars(90, "up"),
		"daily":  buildTrendBars(50, "up"),
		"weekly": buildTrendBars(25, "down"),
	}
	pa := BuildPriceAnalysisFromBars(bars, 20.84, KeyLevelBands{
		SupportLow: 17.77, SupportHigh: 18.23, ResistLow: 22.76, ResistHigh: 23.32,
	}, "")
	if len(pa.Rows) != 4 {
		t.Fatalf("rows=%d", len(pa.Rows))
	}
	if !strings.Contains(pa.SupportResistance, "17.77") || !strings.Contains(pa.SupportResistance, "23.32") {
		t.Fatalf("sr=%q", pa.SupportResistance)
	}
	if pa.Summary == "" {
		t.Fatal("missing summary")
	}
	if pa.SpotLabel != "现价 20.84" {
		t.Fatalf("spot=%q", pa.SpotLabel)
	}
}

func TestSummarizeMultiTimeframeRowsShortBullWeeklyBear(t *testing.T) {
	rows := []PriceAnalysisRow{
		{Timeframe: "5m", Conclusion: RegimeUp},
		{Timeframe: "60m", Conclusion: RegimeUp},
		{Timeframe: "日线", Conclusion: RegimeUp},
		{Timeframe: "周线", Conclusion: RegimeDown},
	}
	s := SummarizeMultiTimeframeRows(rows)
	if !strings.Contains(s, "短中线偏多") || !strings.Contains(s, "周线仍空") {
		t.Fatalf("summary=%q", s)
	}
}

func TestPriceAnalysisJSONPayloadShape(t *testing.T) {
	pa := BuildPriceAnalysisFromBars(map[string][]any{
		"5m": buildTrendBars(90, "up"), "60m": buildTrendBars(90, "up"),
		"daily": buildTrendBars(50, "up"), "weekly": buildTrendBars(25, "down"),
	}, 20.84, KeyLevelBands{SupportLow: 1, ResistHigh: 2}, "")
	raw, err := json.Marshal(pa)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	rows, ok := m["rows"].([]any)
	if !ok || len(rows) != 4 {
		t.Fatalf("rows=%v", m["rows"])
	}
	row0, _ := rows[0].(map[string]any)
	for _, key := range []string{"timeframe", "conclusion", "structure", "bos_choch", "position", "volume_note"} {
		if row0[key] == nil || row0[key] == "" {
			t.Fatalf("missing %s in row", key)
		}
	}
}

func TestAppendPriceAnalysisMarkdownTableHeaders(t *testing.T) {
	pa := BuildPriceAnalysisFromBars(map[string][]any{
		"5m": buildTrendBars(90, "up"), "60m": buildTrendBars(90, "up"),
		"daily": buildTrendBars(50, "up"), "weekly": buildTrendBars(25, "down"),
	}, 10, KeyLevelBands{}, "")
	var b strings.Builder
	AppendAnalysisMarkdown(&b, &pa)
	out := b.String()
	for _, h := range []string{"周期", "结论", "结构", "BOS/CHOCH", "位置", "ER", "RVOL", "量能"} {
		if !strings.Contains(out, h) {
			t.Fatalf("missing header %s in %q", h, out)
		}
	}
}
