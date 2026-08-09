package synthctx

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

// SynthesizerProvider abstracts report.Synthesizer.
type SynthesizerProvider interface {
	Synthesize(ctx context.Context, ws memory.StockWorkspace, evidence []memory.EvidenceRef, mc memory.MarketContext) (report.SynthesisResult, error)
}

type synthesizerContextKey struct{}

// ContextWithSynthesizer attaches an optional report synthesizer to ctx.
func ContextWithSynthesizer(ctx context.Context, synth SynthesizerProvider) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if synth == nil {
		return ctx
	}
	return context.WithValue(ctx, synthesizerContextKey{}, synth)
}

// From returns the synthesizer attached to ctx, if any.
func From(ctx context.Context) SynthesizerProvider {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(synthesizerContextKey{}).(SynthesizerProvider)
	return s
}
