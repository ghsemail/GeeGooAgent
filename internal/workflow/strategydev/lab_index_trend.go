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

// LabIndexTrend is raw multi-TF catalog trend flags from GeeGooSignal lab-api.
type LabIndexTrend struct {
	Code            string                       `json:"code,omitempty"`
	StrategyVersion string                       `json:"strategy_version,omitempty"`
	AnchorTime      string                       `json:"anchor_time,omitempty"`
	Timeframes      map[string]LabIndexTrendTF   `json:"timeframes,omitempty"`
	OK              bool                         `json:"ok"`
	Error           string                       `json:"error,omitempty"`
}

// LabIndexTrendTF is one timeframe snapshot (Signal body + fetch metadata).
type LabIndexTrendTF struct {
	Frequency       string                          `json:"frequency"`
	MonthsBack      int                             `json:"months_back"`
	StrategyVersion string                          `json:"strategy_version,omitempty"`
	BarCount        int                             `json:"bar_count,omitempty"`
	LastBarTime     string                          `json:"last_bar_time,omitempty"`
	Trends          []signalclient.IndexTrendTrendRow `json:"trends,omitempty"`
	Summary         signalclient.IndexTrendSummary  `json:"summary,omitempty"`
	OK              bool                            `json:"ok"`
	Error           string                          `json:"error,omitempty"`
}

// IndexTrendJEV is the JEV 决策 directional read for lab index trend flags.
type IndexTrendJEV struct {
	Source            string                   `json:"source"`
	Model             string                   `json:"model"`
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	RecommendedReason string                   `json:"recommended_reason"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	OK                bool                     `json:"ok"`
	Error             string                   `json:"error"`
}

// LabIndexTrendClient fetches catalog trend flags from lab-api.
type LabIndexTrendClient interface {
	FetchIndexTrend(ctx context.Context, req signalclient.IndexTrendRequest) (signalclient.IndexTrendResponse, error)
}

func fetchLabIndexTrend(
	ctx context.Context,
	client LabIndexTrendClient,
	in LabDetailInput,
	asOf *time.Time,
) *LabIndexTrend {
	out := fetchLabIndexTrendAt(ctx, client, in, asOf)
	if out.OK {
		return out
	}
	if asOf != nil && !asOf.IsZero() && labCodeLooksCrypto(in.Code) {
		if retry := fetchLabIndexTrendAt(ctx, client, in, nil); retry.OK {
			return retry
		}
	}
	return out
}

func fetchLabIndexTrendAt(
	ctx context.Context,
	client LabIndexTrendClient,
	in LabDetailInput,
	asOf *time.Time,
) *LabIndexTrend {
	out := &LabIndexTrend{
		Code:            strings.TrimSpace(in.Code),
		StrategyVersion: "alpha",
		Timeframes:      map[string]LabIndexTrendTF{},
	}
	if client == nil {
		out.Error = "lab index-trend client unavailable"
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
		req signalclient.IndexTrendRequest
	}
	jobs := make([]tfJob, 0, len(freqs))
	for _, f := range freqs {
		apiFreq := priceanalysis.LabKlineFetchFrequency(f)
		months := monthsBackForLabTechnicalAnalysis(in.Months, chart, f, windowEnd)
		jobs = append(jobs, tfJob{
			key: apiFreq,
			req: signalclient.IndexTrendRequest{
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
			tfOut := LabIndexTrendTF{Frequency: job.key, MonthsBack: job.req.MonthsBack}
			resp, err := client.FetchIndexTrend(ctx, job.req)
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
			tfOut.Trends = resp.Trends
			tfOut.Summary = resp.Summary
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
		out.Error = "index-trend fetch failed for all timeframes"
		return out
	}
	out.OK = true
	out.Error = ""
	return out
}

func labIndexTrendInformative(it *LabIndexTrend) bool {
	if it == nil || !it.OK || len(it.Timeframes) == 0 {
		return false
	}
	for _, tf := range it.Timeframes {
		if tf.OK && len(tf.Trends) > 0 {
			return true
		}
	}
	return false
}

func indexTrendStanceLabelCN(stance string) string {
	switch strings.TrimSpace(strings.ToLower(stance)) {
	case "long":
		return "多"
	case "short":
		return "空"
	case "flat":
		return "平"
	default:
		if s := strings.TrimSpace(stance); s != "" {
			return s
		}
		return "未知"
	}
}

func appendLabIndexTrendTF(b *strings.Builder, label string, tf LabIndexTrendTF) {
	fmt.Fprintf(b, "\n[%s]\n", label)
	if tf.LastBarTime != "" {
		fmt.Fprintf(b, "末根 %s；", tf.LastBarTime)
	}
	fmt.Fprintf(b, "汇总 多=%d 空=%d 平=%d net=%d\n", tf.Summary.Long, tf.Summary.Short, tf.Summary.Flat, tf.Summary.Net)
	for _, row := range tf.Trends {
		fmt.Fprintf(b, "- %s(%s)=%s\n", strings.TrimSpace(row.Name), strings.TrimSpace(row.Index), indexTrendStanceLabelCN(row.Stance))
	}
}
