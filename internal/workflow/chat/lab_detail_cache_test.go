package chat

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestLabEpisodeDetailCacheRoundTrip(t *testing.T) {
	chat := &chatsession.ChatSession{ID: "s1", Metadata: map[string]any{}}
	runID := "flow-20261024T120000"
	ref := strategydev.LabDetailRef("buy", 0)
	want := strategydev.LabEpisodeDetail{
		Ref: ref, Side: "buy", BarIndex: 0, Close: 42,
		PriceAnalysis: &strategydev.PriceAnalysis{Summary: "偏多"},
		PriceAnalysisJEV: &strategydev.PriceAnalysisJEV{OK: true, RecommendedChoice: "看多"},
	}
	StoreLabEpisodeDetail(chat, nil, runID, ref, want)
	got, ok := CachedLabEpisodeDetail(chat, runID, ref)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Close != 42 || got.PriceAnalysisJEV.RecommendedChoice != "看多" {
		t.Fatalf("unexpected detail %+v", got)
	}
	export := LabEpisodeDetailsForRun(chat, runID)
	if len(export) != 1 || export[ref].Close != 42 {
		t.Fatalf("export map=%+v", export)
	}
}

func TestLabEpisodeDetailCacheExpires(t *testing.T) {
	ent := labDiagnoseEpisodeDetailsEntry{
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
		Details: map[string]strategydev.LabEpisodeDetail{
			"buy:0": {Ref: "buy:0"},
		},
	}
	raw, _ := json.Marshal(ent)
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			MetaKeyLabDiagnoseEpisodeDetails: map[string]json.RawMessage{"run-old": raw},
		},
	}
	if _, ok := CachedLabEpisodeDetail(chat, "run-old", "buy:0"); ok {
		t.Fatal("expected expired cache miss")
	}
}
