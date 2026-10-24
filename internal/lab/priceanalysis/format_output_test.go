package priceanalysis

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestRoundLabRatio3(t *testing.T) {
	got := RoundLabRatio3(0.28571428571428864)
	if got != 0.286 {
		t.Fatalf("got %v", got)
	}
}

func TestRoundPriceAnalysisForOutputRowsAndTFScores(t *testing.T) {
	pa := PriceAnalysis{
		PriceScore: 12.3456,
		Rows: []PriceAnalysisRow{
			{Timeframe: "1小时", ER: 0.28571428571428864, RVOL: 1.23456789},
		},
		TFScores: map[string]TFScoreBreakdown{
			"1小时": {Subscore: 1.1111111, Conclusion: 0.5555555, Position: 0.3333333, BosChoch: 0.2222222},
		},
		SpotPrice: 123.456789,
	}
	out := RoundPriceAnalysisForOutput(pa)
	if out.Rows[0].ER != 0.286 || out.Rows[0].RVOL != 1.235 {
		t.Fatalf("rows er/rvol=%v %v", out.Rows[0].ER, out.Rows[0].RVOL)
	}
	if out.SpotPrice != pa.SpotPrice {
		t.Fatal("spot price must not be rounded in output helper")
	}
	br := out.TFScores["1小时"]
	if br.Subscore != 1.111 || br.Conclusion != 0.556 {
		t.Fatalf("tf breakdown=%+v", br)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "285714") {
		t.Fatalf("long er in json: %s", s)
	}
}

func TestRoundPriceAnalysisForOutputDoesNotAffectInternalCopy(t *testing.T) {
	er := 0.28571428571428864
	pa := PriceAnalysis{Rows: []PriceAnalysisRow{{ER: er}}}
	_ = RoundPriceAnalysisForOutput(pa)
	if pa.Rows[0].ER != er {
		t.Fatal("RoundPriceAnalysisForOutput must copy rows")
	}
	if math.Abs(pa.Rows[0].ER-er) > 1e-15 {
		t.Fatal("input row unchanged")
	}
}
