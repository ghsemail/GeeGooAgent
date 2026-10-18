package lab

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

func TestRunPriceAnalysisUsesMockLabAPI(t *testing.T) {
	mock := signalclient.NewMockClient()
	mock.BarsFn = func(req signalclient.KlineJobRequest) ([]any, error) {
		n := 120
		if req.Frequency == "weekly" {
			n = 30
		}
		bars := make([]any, n)
		for i := 0; i < n; i++ {
			c := 10.0 + float64(i)*0.01
			bars[i] = map[string]any{
				"time": "2026-01-01 10:00:00", "close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
			}
		}
		return bars, nil
	}
	svc := NewService(mock)
	resp, err := svc.RunPriceAnalysis(context.Background(), PriceAnalysisRequest{
		Code: "00700.HK", SignalFrequency: "60m", Months: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Experiment != ExperimentPriceAnalysis {
		t.Fatalf("experiment=%q", resp.Experiment)
	}
	if len(resp.Analysis.Rows) == 0 {
		t.Fatal("expected analysis rows")
	}
	if resp.Analysis.PriceScore == 0 && resp.Analysis.ScoreReason == "" {
		t.Fatal("expected composite analysis score")
	}
}

func TestRunPriceAnalysisValidation(t *testing.T) {
	svc := &Service{}
	_, err := svc.RunPriceAnalysis(context.Background(), PriceAnalysisRequest{})
	if !IsBadRequest(err) {
		t.Fatalf("err=%v", err)
	}
}
