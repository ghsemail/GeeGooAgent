package memory

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
)

// AdapterConfig wires existing backends into the Memory port.
type AdapterConfig struct {
	Compressor *prompt.Compressor
	Sessions   chatsession.SessionStore
	Evidence   *EvidenceStore
}

// Adapter implements memport.Port using Compressor + SessionStore + EvidenceStore.
type Adapter struct {
	compressor *prompt.Compressor
	sessions   chatsession.SessionStore
	evidence   *EvidenceStore
}

// NewAdapter builds a Memory port from current Go implementations.
func NewAdapter(cfg AdapterConfig) *Adapter {
	return &Adapter{
		compressor: cfg.Compressor,
		sessions:   cfg.Sessions,
		evidence:   cfg.Evidence,
	}
}

// SetCompressor updates the compaction backend (e.g. after RebuildGateway).
func (a *Adapter) SetCompressor(c *prompt.Compressor) {
	if a == nil {
		return
	}
	a.compressor = c
}

// Recall dispatches by kind. Session recall searches past chat sessions.
func (a *Adapter) Recall(ctx context.Context, q memport.RecallQuery) (memport.RecallResult, error) {
	_ = ctx
	switch q.Kind {
	case memport.RecallSession, "":
		return a.recallSessions(q)
	case memport.RecallEvidence:
		return a.recallEvidence(q)
	default:
		return memport.RecallResult{}, fmt.Errorf("memory: unsupported recall kind %q", q.Kind)
	}
}

func (a *Adapter) recallSessions(q memport.RecallQuery) (memport.RecallResult, error) {
	if a == nil || a.sessions == nil {
		return memport.RecallResult{}, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 5
	}
	scan := q.ScanLimit
	if scan <= 0 {
		scan = 30
	}
	hits, err := chatsession.SearchPastSessions(a.sessions, q.Query, q.ExcludeSessionID, limit, scan)
	if err != nil {
		return memport.RecallResult{}, err
	}
	out := memport.RecallResult{
		Hits: make([]memport.RecallHit, 0, len(hits)),
		Data: chatsession.HitsToData(hits),
	}
	for _, h := range hits {
		out.Hits = append(out.Hits, memport.RecallHit{
			ID: h.SessionID, Score: h.Score, Snippet: h.Snippet,
			Data: map[string]any{
				"session_id": h.SessionID, "updated_at": h.UpdatedAt, "score": h.Score,
				"snippet": h.Snippet, "user_queries": h.UserQueries, "stock_events": h.StockEvents,
			},
		})
	}
	return out, nil
}

func (a *Adapter) recallEvidence(q memport.RecallQuery) (memport.RecallResult, error) {
	if a == nil || a.evidence == nil || q.RunID == "" {
		return memport.RecallResult{}, nil
	}
	refs, err := a.evidence.QueryByRun(q.RunID)
	if err != nil {
		return memport.RecallResult{}, err
	}
	out := memport.RecallResult{Hits: make([]memport.RecallHit, 0, len(refs))}
	for _, ref := range refs {
		out.Hits = append(out.Hits, memport.RecallHit{
			ID: ref.ID, Snippet: ref.Summary,
			Data: map[string]any{
				"id": ref.ID, "tool": ref.Tool, "source": ref.Source,
				"summary": ref.Summary, "observed_at": ref.ObservedAt,
			},
		})
	}
	out.Data = map[string]any{"count": len(out.Hits), "refs": out.Hits}
	return out, nil
}

// Store persists auxiliary records (evidence). Conversation SSOT is not written here.
func (a *Adapter) Store(ctx context.Context, rec memport.Record) error {
	_ = ctx
	if a == nil {
		return nil
	}
	switch rec.Kind {
	case memport.RecordEvidence:
		if a.evidence == nil {
			return fmt.Errorf("memory: evidence store not configured")
		}
		return a.evidence.Record(EvidenceRef(rec.Ref), rec.Payload)
	default:
		return fmt.Errorf("memory: unsupported record kind %q", rec.Kind)
	}
}

// Compress runs Hermes-style compaction via prompt.Compressor.
func (a *Adapter) Compress(ctx context.Context, in memport.CompressInput) (memport.CompressOutput, error) {
	out := memport.CompressOutput{
		Messages:             in.Messages,
		PreviousSummary:      in.PreviousSummary,
		EstimatedTokensAfter: in.EstimatedTokens,
	}
	if a == nil || a.compressor == nil || len(in.Messages) == 0 {
		return out, nil
	}
	est := in.EstimatedTokens
	if est <= 0 {
		est = prompt.EstimateTokens(in.Messages)
	}
	var (
		msgs    []llm.Message
		did     bool
		summary string
		err     error
	)
	if in.Hygiene {
		if !a.compressor.ShouldHygiene(est, len(in.Messages)) {
			return out, nil
		}
		msgs, did, summary, err = a.compressor.CompressHygiene(ctx, in.Messages, in.PreviousSummary, est)
	} else {
		if !a.compressor.ShouldCompress(est, len(in.Messages)) {
			return out, nil
		}
		msgs, did, summary, err = a.compressor.Compress(ctx, in.Messages, in.PreviousSummary, est)
	}
	if err != nil || !did {
		return out, nil
	}
	out.Messages = msgs
	out.DidCompress = true
	out.PreviousSummary = summary
	out.EstimatedTokensAfter = prompt.EstimateTokens(msgs)
	return out, nil
}
