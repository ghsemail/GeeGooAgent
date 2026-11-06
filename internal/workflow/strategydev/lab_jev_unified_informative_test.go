package strategydev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

func TestLabJEVUnifiedPlanTANotParticipatingWhenNotInformative(t *testing.T) {
	cli := signalclient.NewHTTPClient("http://unused", "k")
	ta := &LabTechnicalAnalysis{OK: false, Error: "technical-analysis fetch failed for all timeframes"}
	out := &LabEpisodeDetail{TechnicalAnalysis: ta}
	plan := labJEVUnifiedPlanFor(LabDetailInput{TAClient: cli}, out)
	if plan.attemptTA {
		t.Fatalf("expected TA not to participate when raw TA is not informative, plan=%+v", plan)
	}
}

func TestLabJEVUnifiedPlanIndexTrendNotParticipatingWhenNotInformative(t *testing.T) {
	cli := signalclient.NewHTTPClient("http://unused", "k")
	it := &LabIndexTrend{OK: false, Error: "index-trend fetch failed for all timeframes"}
	out := &LabEpisodeDetail{IndexTrend: it}
	plan := labJEVUnifiedPlanFor(LabDetailInput{IndexTrendClient: cli}, out)
	if plan.attemptIndexTrend {
		t.Fatalf("expected index_trend not to participate when raw data is not informative, plan=%+v", plan)
	}
}

func TestApplyLabJEVUnifiedEmittedWhenTAAllFetchFailButPAOK(t *testing.T) {
	var jevCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			jevCalls++
			mockDecisionChoiceResponse(w, map[string]any{
				"direction": map[string]any{
					"type":          "choice",
					"choice":        "看多",
					"probabilities": map[string]float64{"看多": 0.85, "看空": 0.1, "不明确": 0.05},
					"confidence":    0.85,
				},
			})
		case "/v1/decision/chat":
			mockDecisionChatResponse(w, "价格结构偏多主导。")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	jev := NewJevClient(JevConfig{BaseURL: srv.URL, APIKey: "k"})
	detail := LabEpisodeDetail{
		PriceAnalysis: &PriceAnalysis{Code: "000001.SH", Rows: []PriceAnalysisRow{{Timeframe: "日线", Conclusion: "偏多"}}},
		PriceAnalysisJEV: &PriceAnalysisJEV{
			OK: true, RecommendedChoice: "看多", RecommendedIndex: 0, RecommendedReason: "结构偏多",
			Choices: []PriceAnalysisJEVChoice{
				{Label: "看多", Probability: 0.8},
				{Label: "看空", Probability: 0.1},
				{Label: "不明确", Probability: 0.1},
			},
		},
		TechnicalAnalysis: &LabTechnicalAnalysis{
			OK: false, Error: "technical-analysis fetch failed for all timeframes",
		},
		TechnicalAnalysisJEV: &TechnicalAnalysisJEV{
			OK: false, Error: "technical-analysis fetch failed for all timeframes",
		},
	}
	in := LabDetailInput{
		Jev:      jev,
		TAClient: signalclient.NewHTTPClient("http://unused", "k"),
	}
	applyLabJEVUnified(context.Background(), &detail, "buy", in)

	if jevCalls != 1 {
		t.Fatalf("expected exactly 1 JEV choice (unified only), got %d", jevCalls)
	}
	if detail.JEVUnified == nil || !detail.JEVUnified.OK {
		t.Fatalf("expected jev_unified ok, got %+v", detail.JEVUnified)
	}
	plan := labJEVUnifiedPlanFor(in, &detail)
	if plan.attemptTA {
		t.Fatal("TA should not be a participating unified leg")
	}
	legs := collectLabJEVUnifiedLegs(&detail, plan, "buy", in.SignalFreq, in.LongOnly)
	state := BuildLabJEVUnifiedState(plan, legs, "buy", in.LongOnly)
	for _, p := range []string{"技术指标 JEV (technical_analysis_jev)", "未参与"} {
		if !strings.Contains(state, p) {
			t.Fatalf("expected unified state to contain %q, got:\n%s", p, state)
		}
	}
}
