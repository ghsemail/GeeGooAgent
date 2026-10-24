package klinepattern

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestEnrichEpisodeDetailKlineDisplayLight_OmitsMTF(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	ep := slots.SignalEpisodeDetail{StartIdx: 1}
	EnrichEpisodeDetailKlineDisplayLight(&ep, KlinePatternDisplayInput{
		SignalFreq: "60m", Bars: bars,
	})
	if ep.KlinePatternMTF != nil {
		t.Fatalf("light enrich must not emit MTF shell on cards, got %+v", ep.KlinePatternMTF)
	}
	if ep.KlinePatternThreeBar == nil {
		t.Fatal("expected three_bar")
	}
}

func TestBuildKlinePatternMTFContext_UnsupportedVsMissingNotes(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	in := KlinePatternDisplayInput{
		SignalFreq: "60m",
		Bars:       bars,
		MTF: &KlinePatternMTFBundle{
			SignalFreq: "60m",
			ByFreq: map[string]KlinePatternFreqSeries{
				"daily":  {Freq: "daily", FetchOK: false, Unsupported: true},
				"weekly": {Freq: "weekly", FetchOK: true, Bars: bars},
			},
		},
	}
	ctx := BuildKlinePatternMTFContext(1, in)
	if ctx == nil {
		t.Fatal("nil ctx")
	}
	var dailyNote string
	for _, tf := range ctx.Timeframes {
		if tf.Frequency == "daily" {
			dailyNote = tf.AlignNote
		}
		if tf.Frequency == "weekly" && strings.Contains(tf.AlignNote, "拉数失败") {
			t.Fatalf("weekly note=%q", tf.AlignNote)
		}
	}
	if !strings.Contains(dailyNote, "不支持") {
		t.Fatalf("daily note=%q want 不支持", dailyNote)
	}
}
