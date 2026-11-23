package diagnoseskill

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestApplySkillFactLanes_PriceAnalysisIsObjectWithRows(t *testing.T) {
	res := Result{}
	detail := &strategydev.LabEpisodeDetail{
		KlineJEVProfile: strategydev.LabKlineJEVProfileDailyMacro,
		PriceAnalysis: &strategydev.PriceAnalysis{
			Code: "000001.SH",
			Rows: []strategydev.PriceAnalysisRow{{
				Timeframe: "日线", Conclusion: "横盘", Structure: "LH+HL",
			}},
			Summary: "LH+HL",
		},
		TechnicalAnalysis: &strategydev.LabTechnicalAnalysis{
			OK: true, Timeframes: map[string]strategydev.LabTechnicalAnalysisTF{
				"daily": {Frequency: "daily", OK: true},
			},
		},
		IndexTrend: &strategydev.LabIndexTrend{
			OK: true, Timeframes: map[string]strategydev.LabIndexTrendTF{
				"daily": {Frequency: "daily", OK: true},
			},
		},
		KlinePatternSummary: "无明显形态",
		KlinePattern:        &strategydev.LabKlinePatternDetail{Summary: "无明显形态"},
	}
	applySkillFactLanes(res, detail)
	pa, ok := res["price_analysis"].(strategydev.PriceAnalysis)
	if !ok {
		t.Fatalf("price_analysis type=%T", res["price_analysis"])
	}
	if len(pa.Rows) != 1 || pa.Rows[0].Timeframe != "日线" {
		t.Fatalf("rows=%+v", pa.Rows)
	}
	ta, ok := res["technical_analysis"].(*strategydev.LabTechnicalAnalysis)
	if !ok || ta.Timeframes == nil {
		t.Fatalf("technical_analysis=%T", res["technical_analysis"])
	}
	kp, ok := res["kline_pattern"].(map[string]any)
	if !ok || kp["summary"] != "无明显形态" {
		t.Fatalf("kline_pattern=%v", res["kline_pattern"])
	}
}