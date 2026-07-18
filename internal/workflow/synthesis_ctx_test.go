package workflow_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestContextWithSynthesizerRoundTrip(t *testing.T) {
	var s workflow.SynthesizerProvider = &happySynthesizer{}
	ctx := workflow.ContextWithSynthesizer(context.Background(), s)
	if workflow.SynthesizerFrom(ctx) == nil {
		t.Fatal("expected synthesizer")
	}
	if workflow.SynthesizerFrom(context.Background()) != nil {
		t.Fatal("expected nil without injection")
	}
}
