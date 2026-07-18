package workflow

import "context"

type synthesizerContextKey struct{}

// ContextWithSynthesizer attaches an optional report synthesizer to ctx.
// Runner injects this before executing workflow steps.
func ContextWithSynthesizer(ctx context.Context, synth SynthesizerProvider) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if synth == nil {
		return ctx
	}
	return context.WithValue(ctx, synthesizerContextKey{}, synth)
}

// SynthesizerFrom returns the synthesizer attached to ctx, if any.
func SynthesizerFrom(ctx context.Context) SynthesizerProvider {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(synthesizerContextKey{}).(SynthesizerProvider)
	return s
}
