package strategydev

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestBuildKlineJEVStateIncludesKlinePatterns(t *testing.T) {
	kline := &LabJEVKlineInput{
		Summary:    "锤子(看多)",
		Bias:       &slots.KlinePatternBias{Bias: "看多", Score: 1},
		Hits:       []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
		SignalFreq: "5m",
	}
	state := BuildKlineJEVState(kline, "buy")
	if !strings.Contains(state, "锤子") || !strings.Contains(state, "形态摘要") {
		t.Fatalf("state missing kline: %q", state)
	}
	if !strings.Contains(state, "5m 锚点命中") {
		t.Fatalf("expected signal anchor tag: %q", state)
	}
}

func TestResolveKlineJEVPatternOnly(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	kline := &LabJEVKlineInput{
		Summary: "锤子",
		Bias:    &slots.KlinePatternBias{Bias: "看多"},
		Hits:    []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
	}
	out := ResolveKlineJEV(context.Background(), jev, kline, "sell")
	if !out.OK || out.RecommendedChoice != "看多" {
		t.Fatalf("pattern-only: %+v", out)
	}
}

func TestResolveKlineJEVConflictPicksSideNotUnclear(t *testing.T) {
	srv := jevTestServer(t, "看空")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "jev-1.13.0"})
	kline := &LabJEVKlineInput{
		Bias: &slots.KlinePatternBias{Bias: "看空"},
		Hits: []slots.KlinePatternHit{{LabelCN: "射击之星", Direction: -1}},
	}
	out := ResolveKlineJEV(context.Background(), jev, kline, "buy")
	if !out.OK || out.RecommendedChoice != "看空" {
		t.Fatalf("conflict mock should pick 看空, got %+v", out)
	}
}

func TestApplyKlineJEVOnDetail(t *testing.T) {
	srv := jevTestServer(t, "看多")
	defer srv.Close()
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	detail := LabEpisodeDetail{
		KlinePatternSummary: "锤子(看多)",
		KlinePatternBias:    &slots.KlinePatternBias{Bias: "看多"},
		KlinePatterns:       []slots.KlinePatternHit{{LabelCN: "锤子", Direction: 1}},
	}
	applyKlineJEV(context.Background(), &detail, "buy", jev)
	if detail.KlineJEV == nil || !detail.KlineJEV.OK {
		t.Fatalf("kline_jev: %+v", detail.KlineJEV)
	}
	if detail.PriceAnalysisJEV != nil {
		t.Fatal("pattern-only must not populate price_analysis_jev")
	}
}
