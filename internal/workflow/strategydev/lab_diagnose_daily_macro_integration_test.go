package strategydev

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type mapTradingDayChecker map[string]bool

func (m mapTradingDayChecker) IsExchangeTradingDay(_ context.Context, code, calendarDay string) (bool, error) {
	if m == nil {
		return true, nil
	}
	key := calendarDay + "|" + code
	if v, ok := m[key]; ok {
		return v, nil
	}
	if v, ok := m[calendarDay]; ok {
		return v, nil
	}
	return true, nil
}

type recordingDailyFetcher struct {
	daily []any
	calls int
}

func (f *recordingDailyFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	_ = ctx
	_ = code
	_ = months
	_ = asOf
	if NormalizeFrequency(freq) == "daily" {
		f.calls++
		return f.daily, true
	}
	return []any{map[string]any{"time": "2026-11-03 10:00:00", "close": 50.0}}, true
}

type recordingTAClient struct {
	asOfHits int
	nilHits  int
}

func (c *recordingTAClient) FetchTechnicalAnalysis(ctx context.Context, req signalclient.TechnicalAnalysisRequest) (signalclient.TechnicalAnalysisResponse, error) {
	_ = ctx
	if req.AsOf == nil {
		c.nilHits++
	} else {
		c.asOfHits++
	}
	return signalclient.TechnicalAnalysisResponse{BarCount: 1}, nil
}

func countingToolRunner(nilProbe *int) ToolRunner {
	return func(ctx context.Context, name string, args map[string]any) tools.Result {
		_ = ctx
		if name == "probe_bot_signal_series" {
			if _, has := args["as_of"]; !has && nilProbe != nil {
				(*nilProbe)++
			}
		}
		return tools.Result{Status: tools.StatusOK, Data: map[string]any{
			"bars": []any{map[string]any{"time": "2026-11-16 00:00:00", "close": 1.0}},
		}}
	}
}

func TestBuildLabEpisodeDetail_DailyMacroHistoricalBuyUsesMonitorCutoff(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	nowFn := func() time.Time { return time.Date(2026, 11, 16, 16, 30, 0, 0, loc) }
	daily := []any{
		map[string]any{"time": "2026-11-03 00:00:00", "close": 100.0},
		map[string]any{"time": "2026-11-16 00:00:00", "close": 200.0},
	}
	chart := []any{
		map[string]any{"time": "2026-11-03 10:00:00", "close": 99.0},
		map[string]any{"time": "2026-11-03 11:00:00", "close": 100.0},
	}
	fetcher := &recordingDailyFetcher{daily: daily}
	ta := &recordingTAClient{}
	var nilAsOfProbe int
	runner := countingToolRunner(&nilAsOfProbe)
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Run:             runner,
		Code:            "000300.SH",
		SignalFreq:      "60m",
		Months:          3,
		Bars:            chart,
		ProbeRaw:        map[string]any{"bars": chart},
		LabFetcher:      fetcher,
		TAClient:        ta,
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		Now:             nowFn,
		TradingDayChecker:          mapTradingDayChecker{"2026-11-16": true, "2026-11-03": true},
		RequireTradingDayChecker:   true,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.MonitorQuoteCutoffDay != "2026-11-16" {
		t.Fatalf("cutoff day=%q", detail.MonitorQuoteCutoffDay)
	}
	if detail.MonitorQuoteBarClose != 200 {
		t.Fatalf("bar close=%v", detail.MonitorQuoteBarClose)
	}
	if detail.PriceAnalysis == nil {
		t.Fatal("expected PA")
	}
	if detail.JEVUnified == nil {
		t.Fatal("expected jev_unified placeholder without JEV client wiring")
	}
	if detail.JEVUnified.OK {
		t.Fatalf("unified should be fail-closed without Jev client: %+v", detail.JEVUnified)
	}
	if ta.nilHits != 0 {
		t.Fatalf("ta nil asOf retries=%d", ta.nilHits)
	}
	// pattern_hits backfill via EnsurePatternHitsOnDailyBars may still probe without as_of; OHLCV must stay lab daily (bar close).
}

func TestBuildLabEpisodeDetail_DailyMacroCutoffFail_NoPAOrUnified(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	nowFn := func() time.Time { return time.Date(2026, 11, 16, 10, 0, 0, 0, loc) }
	daily := []any{
		map[string]any{"time": "2026-11-16 00:00:00", "close": 200.0},
	}
	chart := []any{map[string]any{"time": "2026-11-03 10:00:00", "close": 99.0}}
	fetcher := &recordingDailyFetcher{daily: daily}
	ta := &recordingTAClient{}
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Code:            "000300.SH",
		SignalFreq:      "60m",
		Months:          1,
		Bars:            chart,
		LabFetcher:      fetcher,
		TAClient:        ta,
		Jev:             NewJevClient(JevConfig{BaseURL: "http://127.0.0.1:9", APIKey: "k"}),
		KlineJEVProfile:          LabKlineJEVProfileDailyMacro,
		Now:                      nowFn,
		TradingDayChecker:        mapTradingDayChecker{"2026-11-16": true},
		RequireTradingDayChecker: true,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.PriceAnalysis != nil {
		t.Fatal("expected no PA when session incomplete")
	}
	if detail.JEVUnified == nil || detail.JEVUnified.OK {
		t.Fatalf("expected fail-closed jev_unified when cutoff incomplete: %+v", detail.JEVUnified)
	}
	if !strings.Contains(detail.JEVUnified.Error, "degraded") {
		t.Fatalf("unified error=%q", detail.JEVUnified.Error)
	}
	if ta.asOfHits+ta.nilHits > 0 {
		t.Fatal("TA should not be called")
	}
}

func TestBuildLabEpisodeDetail_DailyMacroRejectsNonTradingDayBar(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	nowFn := func() time.Time { return time.Date(2026, 11, 16, 16, 30, 0, 0, loc) }
	daily := []any{
		map[string]any{"time": "2026-11-15 00:00:00", "close": 1.0}, // Sunday placeholder
		map[string]any{"time": "2026-11-16 00:00:00", "close": 200.0},
	}
	var nilProbe int
	detail, err := BuildLabEpisodeDetail(context.Background(), LabDetailInput{
		Run: countingToolRunner(&nilProbe),
		Code: "000300.SH", SignalFreq: "60m", Months: 1,
		Bars: []any{map[string]any{"time": "2026-11-03 10:00:00", "close": 1.0}},
		LabFetcher: &recordingDailyFetcher{daily: daily},
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		Now:             nowFn,
		TradingDayChecker: mapTradingDayChecker{
			"2026-11-15": false,
			"2026-11-16": true,
		},
		RequireTradingDayChecker: true,
	}, LabDetailRef("buy", 0))
	if err != nil {
		t.Fatal(err)
	}
	if detail.MonitorQuoteCutoffDay != "2026-11-16" {
		t.Fatalf("day=%q want 11-16 after skipping non-trading placeholder", detail.MonitorQuoteCutoffDay)
	}
}

func TestLabDetailCacheAcceptable_RejectsPreContractMacroPayload(t *testing.T) {
	stale := LabEpisodeDetail{
		KlineJEVProfile: LabKlineJEVProfileDailyMacro,
		LabDetailCacheContract: 0,
		MonitorQuoteCutoff: "2026-11-16T07:00:00Z",
	}
	if LabDetailCacheAcceptable(stale, LabKlineJEVProfileDailyMacro) {
		t.Fatal("expected stale contract rejection")
	}
	stale.LabDetailCacheContract = LabDetailDailyMacroCacheContract
	stale.MonitorSessionClose = ""
	if LabDetailCacheAcceptable(stale, LabKlineJEVProfileDailyMacro) {
		t.Fatal("expected missing session close rejection")
	}
}