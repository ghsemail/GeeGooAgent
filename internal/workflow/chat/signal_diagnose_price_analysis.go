package chat

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func (r *Runner) attachSignalDiagnosePriceAnalysis(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) {
	if flow == nil {
		return
	}
	code := flow.StockCode
	if code == "" {
		return
	}
	months := flow.MonthsBack
	if months <= 0 {
		months = defaultMonthsBack
	}
	freq := probeFrequency(flow)
	spot := spotCloseFromProbe(flow)
	kl := keyLevelBandsFromFlow(flow)
	run := func(c context.Context, name string, args map[string]any) tools.Result {
		return r.runTool(c, toolCtx, name, args, recordTool)
	}
	built := strategydev.BuildRunPriceAnalysis(ctx, run, code, months, spot, kl, freq)
	flow.PriceAnalysis = &built
}

func spotCloseFromProbe(flow *Flow) float64 {
	if flow == nil || flow.ProbeRaw == nil {
		return 0
	}
	bars, _ := flow.ProbeRaw["bars"].([]any)
	if len(bars) == 0 {
		return 0
	}
	row, _ := bars[len(bars)-1].(map[string]any)
	return slots.FloatAny(row["close"])
}

func keyLevelBandsFromFlow(flow *Flow) strategydev.KeyLevelBands {
	if flow == nil {
		return strategydev.KeyLevelBands{}
	}
	kl := flow.KeyLevels
	if kl.SupportLow > 0 || kl.ResistHigh > 0 {
		return strategydev.KeyLevelBands{
			SupportLow:  kl.SupportLow,
			SupportHigh: kl.SupportHigh,
			ResistLow:   kl.ResistLow,
			ResistHigh:  kl.ResistHigh,
		}
	}
	if flow.ProbeRaw != nil {
		return strategydev.KeyLevelBandsFromProbeData(flow.ProbeRaw)
	}
	return strategydev.KeyLevelBands{}
}
