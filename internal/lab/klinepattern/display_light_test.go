package klinepattern

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestEnrichSignalEpisodeEvalKlineDisplayLight_ThreeBarMatchesPatterns(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
		map[string]any{"time": "2024-01-03", "open": 2, "high": 3, "low": 1.5, "close": 2.8},
	}
	hits := []any{
		nil,
		nil,
		[]any{map[string]any{"name": "ThreeWhiteSoldiers", "direction": 1}},
	}
	in := KlinePatternDisplayInput{
		SignalFreq:  "60m",
		Bars:        bars,
		PatternHits: hits,
	}
	eval := slots.SignalEpisodeEval{
		BuyDetails: []slots.SignalEpisodeDetail{{StartIdx: 2, Complete: true}},
	}
	EnrichSignalEpisodeEvalKlineDisplayLight(&eval, in)
	ep := eval.BuyDetails[0]
	if len(ep.KlinePatterns) != 1 || ep.KlinePatterns[0].Name != "ThreeWhiteSoldiers" {
		t.Fatalf("patterns=%+v", ep.KlinePatterns)
	}
	if ep.KlinePatternThreeBar == nil {
		t.Fatal("expected three_bar window")
	}
	if ep.KlinePatternThreeBar.Bias == nil || ep.KlinePatternThreeBar.Bias.Score <= 0 {
		t.Fatalf("three_bar bias=%+v", ep.KlinePatternThreeBar.Bias)
	}
	if ep.KlinePatternMTF == nil || len(ep.KlinePatternMTF.Timeframes) != 4 {
		t.Fatalf("mtf=%+v", ep.KlinePatternMTF)
	}
	sigTF := ep.KlinePatternMTF.Timeframes[0]
	if sigTF.ThreeBar == nil || len(sigTF.ThreeBar.Bars) != 3 {
		t.Fatalf("signal tf three_bar=%+v", sigTF.ThreeBar)
	}
	anchor := sigTF.ThreeBar.Bars[2]
	if anchor.Omitted || len(anchor.KlinePatterns) != 1 {
		t.Fatalf("anchor bar=%+v", anchor)
	}
	if ep.KlinePatternBias == nil || ep.KlinePatternBias.Score <= 0 {
		t.Fatalf("episode bias=%+v", ep.KlinePatternBias)
	}
}
