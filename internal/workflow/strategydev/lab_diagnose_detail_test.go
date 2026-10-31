package strategydev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type blockingLabBarsFetcher struct {
	block time.Duration
}

func (f blockingLabBarsFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	select {
	case <-time.After(f.block):
	case <-ctx.Done():
	}
	return []any{map[string]any{"time": float64(1_700_000_000), "close": 100.0}}, true
}

func TestLabDetailJEVContextSurvivesPhaseDeadline(t *testing.T) {
	parent := context.Background()
	phaseCtx, phaseCancel := labDetailPhaseContext(parent, 30*time.Millisecond)
	defer phaseCancel()
	time.Sleep(50 * time.Millisecond)
	if phaseCtx.Err() == nil {
		t.Fatal("expected phase context deadline")
	}

	var jevCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jevCalled.Store(true)
		switch r.URL.Path {
		case "/v1/decision/choice":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "message": "ok",
				"data": map[string]any{
					"model": "jev-1.13.0",
					"answers": map[string]any{
						"direction": map[string]any{
							"type": "choice", "choice": "看多",
							"probabilities": map[string]float64{"看多": 0.7, "看空": 0.2, "不明确": 0.1},
							"confidence": 0.7,
						},
					},
				},
			})
		case "/v1/decision/chat":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "message": "ok",
				"data": map[string]any{"model": "jev-1.13.0", "content": "60m 结构偏多。"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	pa := &PriceAnalysis{
		Code: "00700.HK",
		Rows: []PriceAnalysisRow{{Timeframe: "60m", Conclusion: "偏多", Structure: "HL", BosChoch: "-", Position: "中"}},
	}
	jevCtx, jevCancel := labDetailJEVContext(parent, 2*time.Second)
	defer jevCancel()
	out := ResolvePriceAnalysisJEV(jevCtx, jev, pa, "buy", false)
	if !out.OK {
		t.Fatalf("expected JEV ok after phase expiry, err=%q", out.Error)
	}
	if !jevCalled.Load() {
		t.Fatal("expected JEV call")
	}

	staleCtx, staleCancel := context.WithTimeout(parent, 30*time.Millisecond)
	defer staleCancel()
	time.Sleep(50 * time.Millisecond)
	staleOut := ResolvePriceAnalysisJEV(staleCtx, jev, pa, "buy", false)
	if staleOut.OK {
		t.Fatal("shared short deadline should fail JEV")
	}
	if staleOut.Error == "" {
		t.Fatal("expected deadline error on shared context")
	}
}

func TestBuildLabEpisodeDetailJEVAfterSlowPhase(t *testing.T) {
	var jevCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jevCalled.Store(true)
		time.Sleep(80 * time.Millisecond)
		switch r.URL.Path {
		case "/v1/decision/choice":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "message": "ok",
				"data": map[string]any{
					"model": "jev-1.13.0",
					"answers": map[string]any{
						"direction": map[string]any{
							"type": "choice", "choice": "看空",
							"probabilities": map[string]float64{"看多": 0.1, "看空": 0.8, "不明确": 0.1},
							"confidence": 0.8,
						},
					},
				},
			})
		case "/v1/decision/chat":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "message": "ok",
				"data": map[string]any{"model": "jev-1.13.0", "content": "更大周期偏空。"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	bars := []any{map[string]any{"time": float64(1_700_000_000), "close": 50.0}}
	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-1.13.0"})
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code: "00700.HK", SignalFreq: "60m", Months: 1, Bars: bars,
		LabFetcher: blockingLabBarsFetcher{block: 120 * time.Millisecond},
		DetailTimeout: 50 * time.Millisecond,
		JEVTimeout:    3 * time.Second,
		Jev:           jev,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.PriceAnalysisJEV == nil {
		t.Fatal("expected price_analysis_jev")
	}
	if !detail.PriceAnalysisJEV.OK {
		t.Fatalf("expected JEV ok despite slow PA phase, err=%q", detail.PriceAnalysisJEV.Error)
	}
	if !jevCalled.Load() {
		t.Fatal("expected JEV invoked after PA")
	}
	if detail.PriceAnalysisJEV.RecommendedChoice != "看空" {
		t.Fatalf("unexpected choice %+v", detail.PriceAnalysisJEV)
	}
}
