package strategydev

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

type taClientProbe struct {
	withAsOf   int
	withNilAsOf int
}

func (c *taClientProbe) FetchTechnicalAnalysis(ctx context.Context, req signalclient.TechnicalAnalysisRequest) (signalclient.TechnicalAnalysisResponse, error) {
	if req.AsOf == nil {
		c.withNilAsOf++
		return signalclient.TechnicalAnalysisResponse{BarCount: 1}, nil
	}
	c.withAsOf++
	return signalclient.TechnicalAnalysisResponse{}, fmt.Errorf("asof_miss")
}

func TestFetchLabTechnicalAnalysis_DailyMacroNoAsOfNilRetry(t *testing.T) {
	probe := &taClientProbe{}
	in := LabDetailInput{
		Code:            "BTC-USD.CRYPTO",
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
	}
	asOf := time.Date(2026, 11, 16, 23, 59, 0, 0, time.UTC)
	_ = fetchLabTechnicalAnalysis(context.Background(), probe, in, &asOf)
	if probe.withNilAsOf != 0 {
		t.Fatalf("nil asOf retries=%d want 0 for daily_macro", probe.withNilAsOf)
	}
	if probe.withAsOf == 0 {
		t.Fatal("expected at least one asOf call")
	}
}

func TestFetchLabTechnicalAnalysis_CryptoMayRetryWithoutDailyMacro(t *testing.T) {
	probe := &taClientProbe{}
	in := LabDetailInput{Code: "BTC-USD.CRYPTO"}
	asOf := time.Date(2026, 11, 16, 23, 59, 0, 0, time.UTC)
	out := fetchLabTechnicalAnalysis(context.Background(), probe, in, &asOf)
	if !out.OK {
		t.Fatal("expected nil asOf retry success for legacy crypto path")
	}
	if probe.withNilAsOf == 0 {
		t.Fatal("expected nil asOf retry when not daily_macro")
	}
}