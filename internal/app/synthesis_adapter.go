package app

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// synthesizerAdapter bridges report.Synthesizer to workflow.SynthesizerProvider,
// keeping the workflow package free of a report import.
type synthesizerAdapter struct {
	inner *report.Synthesizer
}

func (a *synthesizerAdapter) Synthesize(
	ctx context.Context,
	ws memory.StockWorkspace,
	evidence []memory.EvidenceRef,
	mc memory.MarketContext,
) (string, string, string, error) {
	res, err := a.inner.Synthesize(ctx, ws, evidence, mc)
	if err != nil {
		return "", "", "", err
	}
	return res.Reason, res.Suggestion, res.Summary, nil
}

// newSynthesizerAdapter builds an adapter from an LLM gateway, or returns nil
// when gateway is nil (rule-based report path stays in effect).
func newSynthesizerAdapter(gateway *llm.Gateway, model string) *synthesizerAdapter {
	if gateway == nil {
		return nil
	}
	return &synthesizerAdapter{inner: report.NewSynthesizer(gateway, model)}
}

var _ workflow.SynthesizerProvider = (*synthesizerAdapter)(nil)
