package chat

import (
	"context"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func ensureLabFlowDetailRef(flow *Flow) {
	if flow == nil || !flow.LabMode {
		return
	}
	if flow.LabEpisodeSummaries == nil {
		flow.LabEpisodeSummaries = strategydev.LabCardEpisodeSummaries{}
	}
	if _, ok := flow.LabEpisodeSummaries[strategydev.LabFlowDetailRef]; ok {
		return
	}
	flow.LabEpisodeSummaries[strategydev.LabFlowDetailRef] = strategydev.LabEpisodeCardSummary{
		DetailRef:    strategydev.LabFlowDetailRef,
		KlinePattern: "运行摘要（全窗）",
	}
}

func (r *Runner) prewarmLabKlineMTF(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
	freq string,
	months int,
	bars []any,
) {
	if r == nil || flow == nil || !flow.LabMode || len(bars) == 0 {
		return
	}
	if flow.LabKlineMTF != nil {
		return
	}
	if b := strategydev.CachedLabKlineMTF(toolCtx.SessionID, flow.RunID); b != nil {
		flow.LabKlineMTF = b
		return
	}
	start := time.Now()
	run := func(c context.Context, name string, args map[string]any) tools.Result {
		return r.runTool(c, toolCtx, name, args, recordTool)
	}
	mtf := strategydev.BuildKlinePatternMTFForDiagnoseRequest(true, ctx, run, flow.StockCode, freq, months, bars)
	recordPhaseTiming(flow, "evaluate:lab_mtf_prewarm", time.Since(start))
	if mtf != nil {
		flow.LabKlineMTF = mtf
		strategydev.StoreCachedLabKlineMTF(toolCtx.SessionID, flow.RunID, mtf)
	}
}
