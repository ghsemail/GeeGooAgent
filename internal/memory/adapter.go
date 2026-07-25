package memory

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/memory/semantic"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
)

// SemanticStore recalls pgvector memory chunks for gate fallback.
type SemanticStore interface {
	RecallChunks(ctx context.Context, query, userID, excludeSessionID string, limit int) ([]semantic.Chunk, error)
}

// AdapterConfig wires existing backends into the Memory port.
type AdapterConfig struct {
	Compressor    *prompt.Compressor
	Sessions      chatsession.SessionStore
	Evidence      *EvidenceStore
	Semantic      SemanticStore
	SessionRanker memport.SessionRanker
}

// Adapter implements memport.Port using Compressor + SessionStore + EvidenceStore.
type Adapter struct {
	compressor    *prompt.Compressor
	sessions      chatsession.SessionStore
	evidence      *EvidenceStore
	semantic      SemanticStore
	sessionRanker memport.SessionRanker
}

// NewAdapter builds a Memory port from current Go implementations.
func NewAdapter(cfg AdapterConfig) *Adapter {
	return &Adapter{
		compressor:    cfg.Compressor,
		sessions:      cfg.Sessions,
		evidence:      cfg.Evidence,
		semantic:      cfg.Semantic,
		sessionRanker: cfg.SessionRanker,
	}
}

// SetCompressor updates the compaction backend (e.g. after RebuildGateway).
func (a *Adapter) SetCompressor(c *prompt.Compressor) {
	if a == nil {
		return
	}
	a.compressor = c
}

// SetSessionRanker wires optional recall reordering (e.g. cognition Ranker via Agent).
func (a *Adapter) SetSessionRanker(fn memport.SessionRanker) {
	if a == nil {
		return
	}
	a.sessionRanker = fn
}

// SetSessions updates the session store (must match chat API SSOT).
func (a *Adapter) SetSessions(s chatsession.SessionStore) {
	if a != nil {
		a.sessions = s
	}
}

// SetSemantic wires pgvector recall for gate fallback.
func (a *Adapter) SetSemantic(s SemanticStore) {
	if a != nil {
		a.semantic = s
	}
}

// Recall dispatches by kind. Session recall searches past chat sessions.
func (a *Adapter) Recall(ctx context.Context, q memport.RecallQuery) (memport.RecallResult, error) {
	switch q.Kind {
	case memport.RecallSession, "":
		return a.recallSessions(ctx, q)
	case memport.RecallEvidence:
		return a.recallEvidence(q)
	default:
		return memport.RecallResult{}, fmt.Errorf("memory: unsupported recall kind %q", q.Kind)
	}
}

func (a *Adapter) recallSessions(ctx context.Context, q memport.RecallQuery) (memport.RecallResult, error) {
	if a == nil {
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
	var (
		hits []chatsession.SessionRecallHit
		err  error
	)
	if a.sessions != nil {
		hits, err = chatsession.SearchPastSessions(a.sessions, q.Query, q.ExcludeSessionID, q.UserID, limit, scan)
		if err != nil {
			return memport.RecallResult{}, err
		}
	}
	source := "fts"
	if len(hits) == 0 && a.semantic != nil {
		chunks, vErr := a.semantic.RecallChunks(ctx, q.Query, q.UserID, q.ExcludeSessionID, limit)
		if vErr == nil && len(chunks) > 0 {
			source = "vector"
			hits = semanticChunksToSessionHits(chunks)
		}
	}
	out := memport.RecallResult{
		Hits: make([]memport.RecallHit, 0, len(hits)),
		Data: chatsession.HitsToData(hits),
	}
	if out.Data == nil {
		out.Data = map[string]any{}
	}
	out.Data["recall_source"] = source
	for _, h := range hits {
		out.Hits = append(out.Hits, memport.RecallHit{
			ID: h.SessionID, Score: h.Score, Snippet: h.Snippet,
			Data: map[string]any{
				"session_id": h.SessionID, "updated_at": h.UpdatedAt, "score": h.Score,
				"snippet": h.Snippet, "user_queries": h.UserQueries, "stock_events": h.StockEvents,
			},
		})
	}
	if a.sessionRanker != nil && len(out.Hits) > 1 {
		ranked, err := a.sessionRanker(ctx, out.Hits)
		if err == nil && len(ranked) > 0 {
			out.Hits = ranked
			rankedData := memport.RecallHitsToData(ranked)
			rankedData["recall_source"] = source
			out.Data = rankedData
		}
	}
	return out, nil
}

func semanticChunksToSessionHits(chunks []semantic.Chunk) []chatsession.SessionRecallHit {
	out := make([]chatsession.SessionRecallHit, 0, len(chunks))
	for i, c := range chunks {
		score := len(chunks) - i
		if score < 1 {
			score = 1
		}
		out = append(out, chatsession.SessionRecallHit{
			SessionID: c.SessionID,
			Score:     score,
			Snippet:   c.Content,
		})
	}
	return out
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
