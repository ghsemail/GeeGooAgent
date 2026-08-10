package postmarket

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/synthctx"
)

// PostMarketSynthesizerProvider synthesizes App-card postmarket summaries.
type PostMarketSynthesizerProvider interface {
	SynthesizePostMarketSummaries(
		ctx context.Context,
		ws memory.StockWorkspace,
		draft string,
		sessionBias, vsPreMarket string,
		ruleMarket, ruleTrade, ruleExperience string,
	) (report.PostMarketSynthesisResult, error)
}

// PostMarketSynthesizerFrom returns a postmarket synthesizer when supported.
func PostMarketSynthesizerFrom(ctx context.Context) PostMarketSynthesizerProvider {
	synth := synthctx.From(ctx)
	if synth == nil {
		return nil
	}
	p, ok := synth.(PostMarketSynthesizerProvider)
	if !ok {
		return nil
	}
	return p
}
