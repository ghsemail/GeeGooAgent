package strategydev

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

const (
	// DefaultLabDetailTATimeout caps lab-api technical-analysis fetch (daily + signal TF).
	DefaultLabDetailTATimeout = 45 * time.Second
)

// LabTechnicalAnalysis is raw multi-TF indicators from GeeGooSignal lab-api (strategy alpha).
type LabTechnicalAnalysis struct {
	Code            string                              `json:"code,omitempty"`
	StrategyVersion string                              `json:"strategy_version,omitempty"`
	AnchorTime      string                              `json:"anchor_time,omitempty"`
	Timeframes      map[string]LabTechnicalAnalysisTF   `json:"timeframes,omitempty"`
	OK              bool                                `json:"ok"`
	Error           string                              `json:"error,omitempty"`
}

// LabTechnicalAnalysisTF is one timeframe snapshot (Signal body + fetch metadata).
type LabTechnicalAnalysisTF struct {
	Frequency       string                                    `json:"frequency"`
	MonthsBack      int                                       `json:"months_back"`
	StrategyVersion string                                    `json:"strategy_version,omitempty"`
	BarCount        int                                       `json:"bar_count,omitempty"`
	LastBarTime     string                                    `json:"last_bar_time,omitempty"`
	Indicators      signalclient.TechnicalAnalysisIndicators  `json:"indicators,omitempty"`
	OK              bool                                      `json:"ok"`
	Error           string                                    `json:"error,omitempty"`
}

// TechnicalAnalysisJEV is the JEV 决策 directional read for lab technical analysis (parallel to PA JEV).
type TechnicalAnalysisJEV struct {
	Source            string                   `json:"source"`
	Model             string                   `json:"model"`
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	RecommendedReason string                   `json:"recommended_reason"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	OK                bool                     `json:"ok"`
	Error             string                   `json:"error"`
}

// LabTechnicalAnalysisClient fetches strategy-alpha indicators from lab-api.
type LabTechnicalAnalysisClient interface {
	FetchTechnicalAnalysis(ctx context.Context, req signalclient.TechnicalAnalysisRequest) (signalclient.TechnicalAnalysisResponse, error)
}

func labDetailTAContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultLabDetailTATimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func labTAFrequencies(signalFreq string) []string {
	sig := NormalizeFrequency(signalFreq)
	if sig == "" {
		sig = "60m"
	}
	if sig == "daily" {
		return []string{"daily"}
	}
	return []string{"daily", sig}
}

func monthsBackForLabTechnicalAnalysis(chartMonths int, chartBars []any, freq string, windowEnd time.Time) int {
	if NormalizeFrequency(freq) == "daily" {
		return monthsBackForPerSignalPriceFetch(chartMonths, chartBars, "daily", windowEnd)
	}
	mb, err := priceanalysis.MonthsBackForLabFetch(CapMonthsBack(chartMonths), freq)
	if err != nil {
		return CapMonthsBack(chartMonths)
	}
	return mb
}

func fetchLabTechnicalAnalysis(
	ctx context.Context,
	client LabTechnicalAnalysisClient,
	in LabDetailInput,
	asOf *time.Time,
) *LabTechnicalAnalysis {
	out := &LabTechnicalAnalysis{
		Code:            strings.TrimSpace(in.Code),
		StrategyVersion: "alpha",
		Timeframes:      map[string]LabTechnicalAnalysisTF{},
	}
	if client == nil {
		out.Error = "lab technical-analysis client unavailable"
		return out
	}
	if strings.TrimSpace(in.Code) == "" {
		out.Error = "missing code"
		return out
	}
	if asOf != nil && !asOf.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		out.AnchorTime = asOf.In(loc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
	}
	chart := coalesceLookback(in.LookbackBars, in.Bars)
	var windowEnd time.Time
	if len(chart) > 0 {
		_, windowEnd = barSeriesTimeExtent(chart)
	}
	freqs := labTAFrequencies(in.SignalFreq)
	type tfJob struct {
		key string
		req signalclient.TechnicalAnalysisRequest
	}
	jobs := make([]tfJob, 0, len(freqs))
	for _, f := range freqs {
		apiFreq := priceanalysis.LabKlineFetchFrequency(f)
		months := monthsBackForLabTechnicalAnalysis(in.Months, chart, f, windowEnd)
		jobs = append(jobs, tfJob{
			key: apiFreq,
			req: signalclient.TechnicalAnalysisRequest{
				Code:            in.Code,
				Frequency:       apiFreq,
				StrategyVersion: "alpha",
				MonthsBack:      months,
				AsOf:            asOf,
			},
		})
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			tfOut := LabTechnicalAnalysisTF{Frequency: job.key, MonthsBack: job.req.MonthsBack}
			resp, err := client.FetchTechnicalAnalysis(ctx, job.req)
			if err != nil {
				tfOut.Error = err.Error()
				mu.Lock()
				out.Timeframes[job.key] = tfOut
				mu.Unlock()
				return
			}
			tfOut.OK = true
			tfOut.StrategyVersion = signalclient.NormalizeStrategyVersion(resp.StrategyVersion)
			tfOut.BarCount = resp.BarCount
			tfOut.LastBarTime = resp.LastBarTime
			tfOut.Indicators = resp.Indicators
			mu.Lock()
			out.Timeframes[job.key] = tfOut
			mu.Unlock()
		}()
	}
	wg.Wait()

	okCount := 0
	for _, tf := range out.Timeframes {
		if tf.OK {
			okCount++
		}
	}
	if okCount == 0 {
		out.Error = "technical-analysis fetch failed for all timeframes"
		return out
	}
	out.OK = true
	out.Error = ""
	return out
}

func applyLabTechnicalAnalysis(out *LabEpisodeDetail, ta *LabTechnicalAnalysis, jevCtx context.Context, side string, in LabDetailInput) {
	if out == nil {
		return
	}
	out.TechnicalAnalysis = ta
	rec := ResolveTechnicalAnalysisJEV(jevCtx, in.Jev, ta, side, in.SignalFreq)
	out.TechnicalAnalysisJEV = &rec
}

func labTAInformative(ta *LabTechnicalAnalysis) bool {
	if ta == nil || !ta.OK || len(ta.Timeframes) == 0 {
		return false
	}
	for _, tf := range ta.Timeframes {
		if tf.OK {
			return true
		}
	}
	return false
}

func labTATimeframeLabel(freq string) string {
	switch NormalizeFrequency(freq) {
	case "daily":
		return "日线"
	case "60m":
		return "60m"
	case "5m":
		return "5m"
	case "weekly", "1wk":
		return "周线"
	default:
		if f := strings.TrimSpace(freq); f != "" {
			return f
		}
		return "信号周期"
	}
}

func elderImpulseLabel(ema13Slope, histogramSlope float64) string {
	switch {
	case ema13Slope > 0 && histogramSlope > 0:
		return "绿"
	case ema13Slope < 0 && histogramSlope < 0:
		return "蓝"
	default:
		return "红"
	}
}

func valueZoneLabelCN(zone string) string {
	switch strings.TrimSpace(zone) {
	case "above_value_area":
		return "价值区上方"
	case "below_value_area":
		return "价值区下方"
	case "inside_value_area":
		return "价值区内"
	default:
		if z := strings.TrimSpace(zone); z != "" {
			return z
		}
		return "未知"
	}
}

func forceIndex13Sign(fi float64) string {
	if fi > 0 {
		return "pos"
	}
	if fi < 0 {
		return "neg"
	}
	return "zero"
}

func appendLabTAIndicators(b *strings.Builder, label string, ind signalclient.TechnicalAnalysisIndicators) {
	fmt.Fprintf(b, "\n[%s]\n", label)
	ema := ind.EMA
	fmt.Fprintf(b, "价值区(EMA13/26)：%s；EMA13斜率=%.4g EMA26斜率=%.4g\n",
		valueZoneLabelCN(ema.ValueZone), ema.EMA13Slope, ema.EMA26Slope)
	env := ind.Envelope
	fmt.Fprintf(b, "Envelope：中轨=%.4g 上=%.4g 下=%.4g\n", env.MidEMA22, env.Upper, env.Lower)
	macd := ind.MACD
	imp := elderImpulseLabel(ema.EMA13Slope, macd.HistogramSlope)
	fmt.Fprintf(b, "MACD：柱=%.4g 柱斜率=%.4g；Impulse(EMA13斜率+柱斜率)=%s\n",
		macd.Histogram, macd.HistogramSlope, imp)
	st := ind.Stochastic
	fmt.Fprintf(b, "Stochastic：K=%.2f D=%.2f 区=%s\n", st.SlowK, st.SlowD, strings.TrimSpace(st.Zone))
	rsi := ind.RSI
	fmt.Fprintf(b, "RSI：%.2f 区=%s\n", rsi.Value, strings.TrimSpace(rsi.Zone))
	fi := ind.ForceIndex
	fmt.Fprintf(b, "Force Index：EMA2=%.4g EMA13=%.4g (fi13=%s)\n", fi.EMA2, fi.EMA13, forceIndex13Sign(fi.EMA13))
}
