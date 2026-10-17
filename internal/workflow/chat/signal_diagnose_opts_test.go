package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestParseSignalDiagnoseOptsProbeOverrides(t *testing.T) {
	raw := map[string]any{
		"signal_diagnose": map[string]any{
			"use_price_analysis_episode_filter": true,
			"price_analysis_buy_score_direction": "reverse",
			"price_analysis_gate_score_mode":     "htf_only",
			"frequency":                         "60m",
			"months_back":                3,
			"buy_signal": []any{
				map[string]any{"index": "SAR", "type": "signal"},
			},
			"sell_signal": []any{
				map[string]any{"index": "SAR", "type": "signal"},
			},
		},
	}
	o := ParseSignalDiagnoseOptsFromWorkflowOptions(raw)
	if o == nil {
		t.Fatal("nil opts")
	}
	if o.UsePriceAnalysisEpisodeFilter == nil || !*o.UsePriceAnalysisEpisodeFilter {
		t.Fatal("price filter")
	}
	if o.PriceAnalysisBuyScoreDirection != "reverse" {
		t.Fatalf("buy score direction=%q", o.PriceAnalysisBuyScoreDirection)
	}
	if o.PriceAnalysisGateScoreMode != "htf_only" {
		t.Fatalf("gate score mode=%q", o.PriceAnalysisGateScoreMode)
	}
	if o.Frequency != "60m" || o.MonthsBack != 3 {
		t.Fatalf("freq/months=%q %d", o.Frequency, o.MonthsBack)
	}
	if len(o.BuySignal) != 1 || len(o.SellSignal) != 1 {
		t.Fatalf("buy=%d sell=%d", len(o.BuySignal), len(o.SellSignal))
	}
}

func TestNewSignalDiagnoseFlowSoftDefaults(t *testing.T) {
	flow := newSignalDiagnoseFlow("诊断 SAR+MACD · 00700.HK")
	if flow.UsePriceAnalysisEpisodeFilter {
		t.Fatal("hard episode filter should default off")
	}
	if flow.PriceAnalysisBuyScoreDirection != strategydev.BuyScoreDirectionReverse {
		t.Fatalf("buy score direction=%q", flow.PriceAnalysisBuyScoreDirection)
	}
	if flow.PriceAnalysisGateScoreMode != "" {
		t.Fatalf("gate score mode should default empty/full, got %q", flow.PriceAnalysisGateScoreMode)
	}
}

func TestEffectiveSignalDiagnoseBuyScoreDirection(t *testing.T) {
	if effectiveSignalDiagnoseBuyScoreDirection("") != strategydev.BuyScoreDirectionReverse {
		t.Fatal("empty should reverse")
	}
	if effectiveSignalDiagnoseBuyScoreDirection(strategydev.BuyScoreDirectionNormal) != strategydev.BuyScoreDirectionNormal {
		t.Fatal("explicit normal should stay")
	}
}

func TestParseSellMergeFromWorkflowOptions(t *testing.T) {
	o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
		"signal_diagnose": map[string]any{
			"sell_merge": "or",
			"buy_merge":  "and",
		},
	})
	if o == nil || o.SellMerge != "or" || o.BuyMerge != "and" {
		t.Fatalf("got %+v", o)
	}
}
