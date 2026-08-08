package agent

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ReportSynthesizer runs evidence-only LLM report synthesis through the same
// gateway as the ReAct loop. Implements workflow.SynthesizerProvider when
// passed from app wiring.
type ReportSynthesizer struct {
	inner *report.Synthesizer
	bus   tools.EventEmitter
}

// NewReportSynthesizer creates a report synthesizer backed by gateway.
func NewReportSynthesizer(gateway *llm.Gateway, model string, bus tools.EventEmitter) *ReportSynthesizer {
	if gateway == nil {
		return nil
	}
	return &ReportSynthesizer{
		inner: report.NewSynthesizer(gateway, model),
		bus:   bus,
	}
}

// SetGateway keeps synthesis on the same gateway as Agent.Run after /model.
func (s *ReportSynthesizer) SetGateway(gateway *llm.Gateway) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.SetGateway(gateway)
}

// Available reports whether synthesis can run.
func (s *ReportSynthesizer) Available() bool {
	return s != nil && s.inner != nil && s.inner.Available()
}

// Synthesize generates suggested fields plus reason/suggestion/summary from evidence.
func (s *ReportSynthesizer) Synthesize(
	ctx context.Context,
	ws memory.StockWorkspace,
	evidence []memory.EvidenceRef,
	marketContext memory.MarketContext,
) (report.SynthesisResult, error) {
	if s == nil || s.inner == nil || !s.inner.Available() {
		return report.SynthesisResult{}, fmt.Errorf("report synthesizer not available")
	}
	s.emit("SynthesisStarted", map[string]any{
		"code": ws.Code, "stock_name": ws.StockName, "evidence_count": len(evidence),
	})
	res, err := s.inner.Synthesize(ctx, ws, evidence, marketContext)
	if err != nil {
		s.emit("SynthesisFailed", map[string]any{
			"code": ws.Code, "error": err.Error(),
		})
		return report.SynthesisResult{}, err
	}
	s.emit("SynthesisCompleted", map[string]any{
		"code": ws.Code, "suggested_result": res.SuggestedResult,
		"suggestion": res.Suggestion, "summary_chars": len(res.Summary),
	})
	return res, nil
}

// SynthesizeMarket generates a single-market pre-market report bundle.
func (s *ReportSynthesizer) SynthesizeMarket(
	ctx context.Context,
	market string,
	draft string,
	marketContext memory.MarketContext,
	evidence []memory.EvidenceRef,
	template string,
) (report.MarketSynthesisResult, error) {
	if s == nil || s.inner == nil || !s.inner.Available() {
		return report.MarketSynthesisResult{}, fmt.Errorf("report synthesizer not available")
	}
	s.emit("MarketSynthesisStarted", map[string]any{
		"market": market, "evidence_count": len(evidence),
	})
	res, err := s.inner.SynthesizeMarket(ctx, market, draft, marketContext, evidence, template)
	if err != nil {
		s.emit("MarketSynthesisFailed", map[string]any{
			"market": market, "error": err.Error(),
		})
		return report.MarketSynthesisResult{}, err
	}
	s.emit("MarketSynthesisCompleted", map[string]any{
		"market": market, "result": res.Result, "confidence": res.Confidence, "summary_chars": len(res.Summary),
	})
	return res, nil
}

func (s *ReportSynthesizer) emit(event string, payload map[string]any) {
	if s != nil && s.bus != nil {
		s.bus.Emit(event, payload)
	}
}
