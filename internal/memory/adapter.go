package memory

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
)

// FactsStore searches durable semantic facts (Waku FTS parity).
type FactsStore interface {
	SearchRows(ctx context.Context, userID, query string, topK int) ([]facts.Row, error)
}

// EpisodicStore searches dated episode summaries.
type EpisodicStore interface {
	SearchEpisodes(ctx context.Context, query, userID string, limit int) ([]episodic.Episode, error)
}

// AdapterConfig wires existing backends into the Memory port.
type AdapterConfig struct {
	Compressor    *prompt.Compressor
	Sessions      chatsession.SessionStore
	Evidence      *EvidenceStore
	Facts         FactsStore
	Episodic      EpisodicStore
	SessionRanker memport.SessionRanker
}

// Adapter implements memport.Port using Compressor + SessionStore + EvidenceStore.
type Adapter struct {
	compressor    *prompt.Compressor
	sessions      chatsession.SessionStore
	evidence      *EvidenceStore
	facts         FactsStore
	episodic      EpisodicStore
	sessionRanker memport.SessionRanker
}

// NewAdapter builds a Memory port from current Go implementations.
func NewAdapter(cfg AdapterConfig) *Adapter {
	return &Adapter{
		compressor:    cfg.Compressor,
		sessions:      cfg.Sessions,
		evidence:      cfg.Evidence,
		facts:         cfg.Facts,
		episodic:      cfg.Episodic,
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

// SetFacts wires Waku-style semantic FTS recall.
func (a *Adapter) SetFacts(s FactsStore) {
	if a != nil {
		a.facts = s
	}
}

// SetEpisodic wires episodic recall for the retrieval gate.
func (a *Adapter) SetEpisodic(s EpisodicStore) {
	if a != nil {
		a.episodic = s
	}
}

// Recall dispatches by kind. Session recall searches facts + episodes (Waku parity).
func (a *Adapter) Recall(ctx context.Context, q memport.RecallQuery) (memport.RecallResult, error) {
	switch q.Kind {
	case memport.RecallSession, "":
		return a.recallMemory(ctx, q)
	case memport.RecallEvidence:
		return a.recallEvidence(q)
	default:
		return memport.RecallResult{}, fmt.Errorf("memory: unsupported recall kind %q", q.Kind)
	}
}

func (a *Adapter) recallMemory(ctx context.Context, q memport.RecallQuery) (memport.RecallResult, error) {
	if a == nil {
		return memport.RecallResult{}, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 4
	}
	var hits []memport.RecallHit
	source := "none"

	if a.facts != nil {
		rows, err := a.facts.SearchRows(ctx, q.UserID, q.Query, limit)
		if err != nil {
			return memport.RecallResult{}, err
		}
		for i, r := range rows {
			score := len(rows) - i
			if score < 1 {
				score = 1
			}
			hits = append(hits, memport.RecallHit{
				ID:      fmt.Sprintf("fact:%d", r.ID),
				Score:   score,
				Snippet: facts.Format(r.Subject, r.Content),
				Data: map[string]any{
					"kind": "fact", "id": r.ID, "subject": r.Subject, "content": r.Content,
				},
			})
		}
		if len(rows) > 0 {
			source = "facts"
		}
	}

	if a.episodic != nil {
		eps, err := a.episodic.SearchEpisodes(ctx, q.Query, q.UserID, 3)
		if err != nil {
			return memport.RecallResult{}, err
		}
		for i, ep := range eps {
			score := len(eps) - i
			if score < 1 {
				score = 1
			}
			snippet := episodic.Format(ep.HappenedAt, ep.Summary)
			hits = append(hits, memport.RecallHit{
				ID:      fmt.Sprintf("episode:%d", ep.ID),
				Score:   score,
				Snippet: snippet,
				Data: map[string]any{
					"kind": "episode", "id": ep.ID, "session_id": ep.SessionID,
				},
			})
		}
		if len(eps) > 0 {
			if source == "none" {
				source = "episodic"
			} else {
				source = "facts+episodic"
			}
		}
	}

	out := memport.RecallResult{
		Hits: make([]memport.RecallHit, 0, len(hits)),
		Data: map[string]any{"recall_source": source},
	}
	for _, h := range hits {
		out.Hits = append(out.Hits, h)
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
