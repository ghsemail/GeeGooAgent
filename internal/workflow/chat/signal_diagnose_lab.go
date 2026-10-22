package chat

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab"
	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const labPriceSummaryTimeout = 4 * time.Second

func patternHitsFromProbe(probeRaw map[string]any) []any {
	if probeRaw == nil {
		return nil
	}
	hits, _ := probeRaw["pattern_hits"].([]any)
	return hits
}

func (r *Runner) attachSignalDiagnoseLabPriceSummary(ctx context.Context, flow *Flow, toolCtx tools.Context) {
	if flow == nil || !flow.LabMode || r == nil || r.LabConfig == nil {
		return
	}
	code := strings.TrimSpace(flow.StockCode)
	if code == "" {
		return
	}
	months := flow.MonthsBack
	if months <= 0 {
		months = defaultMonthsBack
	}
	freq := probeFrequency(flow)
	spot := spotCloseFromProbe(flow)
	fetcher := lab.SignalBarsFetcher{
		Client: signalclient.NewHTTPClient(r.LabConfig.LabAPIURL, r.LabConfig.LabAPIKey),
	}
	ctx, cancel := context.WithTimeout(ctx, labPriceSummaryTimeout)
	defer cancel()
	pa := priceanalysis.Run(ctx, fetcher, priceanalysis.RunRequest{
		Code: code, Months: strategydev.CapMonthsBack(months), SignalFreq: freq, SpotPrice: spot,
		KeyLevels: keyLevelsToLabPrice(keyLevelBandsFromFlow(flow)),
	})
	pa = priceanalysis.RoundPriceAnalysisForOutput(pa)
	flow.LabFlowPriceSummary = strategydev.PriceAnalysisSummaryLine(labPriceToStrategy(&pa))
}

func keyLevelsToLabPrice(kl strategydev.KeyLevelBands) priceanalysis.KeyLevelBands {
	return priceanalysis.KeyLevelBands{
		SupportLow: kl.SupportLow, SupportHigh: kl.SupportHigh,
		ResistLow: kl.ResistLow, ResistHigh: kl.ResistHigh,
	}
}

func tfScoresFromLabPrice(in map[string]priceanalysis.TFScoreBreakdown) map[string]strategydev.TFScoreBreakdown {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]strategydev.TFScoreBreakdown, len(in))
	for k, v := range in {
		out[k] = strategydev.TFScoreBreakdown{
			Subscore: v.Subscore, Conclusion: v.Conclusion, Position: v.Position,
			BosChoch: v.BosChoch, Missing: v.Missing,
		}
	}
	return out
}

func labPriceToStrategy(pa *priceanalysis.PriceAnalysis) *strategydev.PriceAnalysis {
	if pa == nil {
		return nil
	}
	out := &strategydev.PriceAnalysis{
		Code: pa.Code, SpotPrice: pa.SpotPrice, SpotLabel: pa.SpotLabel,
		SupportResistance: pa.SupportResistance, Summary: pa.Summary,
		FilterTimeframe: pa.FilterTimeframe, FilterTimeframeNote: pa.FilterTimeframeNote,
		AnchorTime: pa.AnchorTime, AnchorClose: pa.AnchorClose,
		PriceScore: pa.PriceScore, TFScores: tfScoresFromLabPrice(pa.TFScores), ScoreReason: pa.ScoreReason,
	}
	for _, row := range pa.Rows {
		out.Rows = append(out.Rows, strategydev.PriceAnalysisRow{
			Timeframe: row.Timeframe, Conclusion: row.Conclusion, Structure: row.Structure,
			BosChoch: row.BosChoch, Position: row.Position, ER: row.ER, RVOL: row.RVOL,
			VolumeNote: row.VolumeNote,
		})
	}
	return out
}

// LabConfig supplies GeeGooSignal lab-api endpoints for signal-diagnose lab_mode.
type LabConfig struct {
	LabAPIURL string
	LabAPIKey string
}
