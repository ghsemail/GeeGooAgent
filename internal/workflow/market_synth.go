package workflow

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

// MarketSynthesizerProvider synthesizes a single-market pre-market report.
type MarketSynthesizerProvider interface {
	SynthesizeMarket(
		ctx context.Context,
		market string,
		draft string,
		marketContext memory.MarketContext,
		evidence []memory.EvidenceRef,
		template string,
	) (report.MarketSynthesisResult, error)
}

// MarketSynthesizerFrom returns a market synthesizer when the injected provider supports it.
func MarketSynthesizerFrom(ctx context.Context) MarketSynthesizerProvider {
	synth := SynthesizerFrom(ctx)
	if synth == nil {
		return nil
	}
	m, ok := synth.(MarketSynthesizerProvider)
	if !ok {
		return nil
	}
	return m
}
