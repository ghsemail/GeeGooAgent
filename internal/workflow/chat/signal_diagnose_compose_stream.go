package chat

import (
	"context"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const composeStreamPublishInterval = 300 * time.Millisecond

type composeStreamState struct {
	started time.Time
	mu      sync.Mutex
	acc     llm.AnswerStreamAccumulator
	sections llm.ComposeSectionRenderer
	chunks   int64
	reasoning int64
	firstReasoningMs int64
	firstContentMs   int64
}

func newComposeStreamState() *composeStreamState {
	s := &composeStreamState{started: time.Now()}
	s.acc.OnVisible = func(chunk string) {
		s.mu.Lock()
		s.sections.PushModelText(chunk)
		s.mu.Unlock()
	}
	return s
}

func (s *composeStreamState) ingestDelta(delta llm.StreamDelta, appendReasoning func(string)) {
	if rc := delta.ReasoningContent; rc != "" {
		s.mu.Lock()
		s.chunks++
		s.reasoning += int64(len([]rune(rc)))
		if s.firstReasoningMs == 0 {
			s.firstReasoningMs = time.Since(s.started).Milliseconds()
		}
		s.mu.Unlock()
		appendReasoning(rc)
		return
	}
	if delta.Content == "" {
		return
	}
	s.mu.Lock()
	s.chunks++
	if s.firstContentMs == 0 {
		s.firstContentMs = time.Since(s.started).Milliseconds()
	}
	s.mu.Unlock()
	s.acc.PushContent(delta.Content)
}

func (s *composeStreamState) snapshot() (doc string, chunks, reasoning, firstR, firstC int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sections.Document(), s.chunks, s.reasoning, s.firstReasoningMs, s.firstContentMs
}

func (s *composeStreamState) applyMetrics(flow *Flow) {
	if flow == nil || s == nil {
		return
	}
	_, chunks, reasoning, firstR, firstC := s.snapshot()
	flow.ComposeChunks = chunks
	flow.ComposeReasoningChars = reasoning
	flow.ComposeFirstReasoningMs = firstR
	flow.ComposeFirstContentMs = firstC
	recordPhaseGauge(flow, "compose_chunks", chunks)
	recordPhaseGauge(flow, "compose_reasoning_chars", reasoning)
	if firstR > 0 {
		recordPhaseGauge(flow, "compose_first_reasoning_ms", firstR)
	}
	if firstC > 0 {
		recordPhaseGauge(flow, "compose_first_content_ms", firstC)
	}
}

type composeStreamPublisher struct {
	r     *Runner
	flow  *Flow
	state *composeStreamState
	stop  context.CancelFunc
	wg    sync.WaitGroup
}

func startComposeStreamPublisher(parent context.Context, r *Runner, flow *Flow, state *composeStreamState) *composeStreamPublisher {
	if r == nil || flow == nil || state == nil {
		return nil
	}
	pubCtx, cancel := context.WithCancel(parent)
	p := &composeStreamPublisher{r: r, flow: flow, state: state, stop: cancel}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(composeStreamPublishInterval)
		defer ticker.Stop()
		for {
			select {
			case <-pubCtx.Done():
				return
			case <-ticker.C:
				p.flush(false)
			}
		}
	}()
	return p
}

func (p *composeStreamPublisher) flush(final bool) {
	if p == nil {
		return
	}
	doc, _, _, _, _ := p.state.snapshot()
	if doc == "" && !final {
		return
	}
	p.flow.EvalJudgment = doc
	p.flow.CardStage = "compose_stream"
	p.r.emitCard(nil, p.flow)
	if p.r.OnProgress != nil {
		p.r.OnProgress("eval_judgment_delta", map[string]any{
			"run_id": p.flow.RunID, "partial": doc,
		})
	}
}

func (p *composeStreamPublisher) finish() {
	if p == nil {
		return
	}
	if p.stop != nil {
		p.stop()
	}
	p.wg.Wait()
	p.flush(true)
	p.state.applyMetrics(p.flow)
}
