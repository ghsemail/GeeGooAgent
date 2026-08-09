package workflow

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

// IntradaySynthesizerProvider synthesizes intraday summary and reason.
type IntradaySynthesizerProvider interface {
	SynthesizeIntraday(
		ctx context.Context,
		ws memory.StockWorkspace,
		draft string,
		result, confidence string,
	) (report.IntradaySynthesisResult, error)
}

// IntradaySynthesizerFrom returns an intraday synthesizer from context, if wired.
func IntradaySynthesizerFrom(ctx context.Context) IntradaySynthesizerProvider {
	synth := SynthesizerFrom(ctx)
	if synth == nil {
		return nil
	}
	p, ok := synth.(IntradaySynthesizerProvider)
	if !ok {
		return nil
	}
	return p
}
