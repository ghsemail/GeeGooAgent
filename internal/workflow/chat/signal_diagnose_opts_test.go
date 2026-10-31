package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
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
	if flow.TradeDirection != runtime.TradeDirectionBoth {
		t.Fatalf("trade_direction=%q want both", flow.TradeDirection)
	}
	if flow.UsePriceAnalysisEpisodeFilter {
		t.Fatal("hard episode filter should default off")
	}
	if flow.PriceAnalysisBuyScoreDirection != strategydev.BuyScoreDirectionReverse {
		t.Fatalf("buy score direction=%q", flow.PriceAnalysisBuyScoreDirection)
	}
	if flow.PriceAnalysisGateScoreMode != strategydev.PriceAnalysisGateScoreModeHTFOnly {
		t.Fatalf("gate score mode should soft-default htf_only, got %q", flow.PriceAnalysisGateScoreMode)
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

func TestParseTradeDirectionFromWorkflowOptions(t *testing.T) {
	t.Run("omitted defaults unset pointer", func(t *testing.T) {
		o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
			"signal_diagnose": map[string]any{},
		})
		if o == nil || o.TradeDirection != nil {
			t.Fatalf("TradeDirection=%v want nil", o)
		}
	})
	t.Run("long_only", func(t *testing.T) {
		o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
			"signal_diagnose": map[string]any{"trade_direction": " LONG_ONLY "},
		})
		if o == nil || o.TradeDirection == nil || *o.TradeDirection != runtime.TradeDirectionLongOnly {
			t.Fatalf("got %+v", o)
		}
	})
	t.Run("both explicit", func(t *testing.T) {
		o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
			"signal_diagnose": map[string]any{"trade_direction": "both"},
		})
		if o == nil || o.TradeDirection == nil || *o.TradeDirection != runtime.TradeDirectionBoth {
			t.Fatalf("got %+v", o)
		}
	})
	t.Run("invalid maps to both", func(t *testing.T) {
		o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
			"signal_diagnose": map[string]any{"trade_direction": "nope"},
		})
		if o == nil || o.TradeDirection == nil || *o.TradeDirection != runtime.TradeDirectionBoth {
			t.Fatalf("got %+v", o)
		}
	})
}

func TestApplyInboundPanelOpsTradeDirection(t *testing.T) {
	td := runtime.TradeDirectionLongOnly
	session := &runtime.Session{InboundPanelOps: &runtime.SignalDiagnoseOpts{TradeDirection: &td}}
	flow := newSignalDiagnoseFlow("诊断 x · y")
	ApplyInboundPanelOpsToFlow(session, flow)
	if flow.TradeDirection != runtime.TradeDirectionLongOnly {
		t.Fatalf("flow=%q", flow.TradeDirection)
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
