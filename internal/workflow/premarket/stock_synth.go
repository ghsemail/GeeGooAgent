package premarket

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/synthctx"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

// StockPreMarketSynthesizerProvider synthesizes a full stock pre-market report.
type StockPreMarketSynthesizerProvider interface {
	SynthesizeStockPreMarket(
		ctx context.Context,
		ws memory.StockWorkspace,
		draft string,
		evidence []memory.EvidenceRef,
		marketContext memory.MarketContext,
		marketReportSummary, template string,
	) (report.StockPreMarketSynthesisResult, error)
}

// StockPreMarketSynthesizerFrom returns a stock pre-market synthesizer when supported.
func StockPreMarketSynthesizerFrom(ctx context.Context) StockPreMarketSynthesizerProvider {
	synth := synthctx.From(ctx)
	if synth == nil {
		return nil
	}
	s, ok := synth.(StockPreMarketSynthesizerProvider)
	if !ok {
		return nil
	}
	return s
}
