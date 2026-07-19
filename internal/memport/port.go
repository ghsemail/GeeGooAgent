package memport

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// RecallKind selects which memory subsystem to query.
type RecallKind string

const (
	RecallSession  RecallKind = "session"
	RecallEvidence RecallKind = "evidence"
)

// RecordKind selects what to persist via Store.
type RecordKind string

const (
	RecordEvidence RecordKind = "evidence"
)

// RecallQuery is a read-only memory lookup.
type RecallQuery struct {
	Kind             RecallKind
	Query            string
	SessionID        string
	ExcludeSessionID string
	Limit            int
	ScanLimit        int
	RunID            string
}

// RecallHit is one ranked memory match.
type RecallHit struct {
	ID      string
	Score   int
	Snippet string
	Data    map[string]any
}

// RecallResult aggregates recall hits; Data is tool/API friendly when set.
type RecallResult struct {
	Hits []RecallHit
	Data map[string]any
}

// Record is a write intent. Session conversation truth stays in chatsession SSOT.
type Record struct {
	Kind    RecordKind
	Ref     EvidenceRef
	Payload any
}

// EvidenceRef is a minimal evidence record for Store (avoids importing memory package).
type EvidenceRef struct {
	ID          string
	RunID       string
	Tool        string
	Source      string
	ObservedAt  string
	PayloadHash string
	Summary     string
}

// CompressInput carries session messages for compaction.
type CompressInput struct {
	SessionID       string
	Messages        []llm.Message
	PreviousSummary string
	EstimatedTokens int
	Hygiene         bool
}

// CompressOutput is the compaction result applied by the Kernel.
type CompressOutput struct {
	Messages             []llm.Message
	DidCompress          bool
	PreviousSummary      string
	EstimatedTokensAfter int
}

// RecallHitsToData builds a tool/API payload from ranked recall hits.
func RecallHitsToData(hits []RecallHit) map[string]any {
	matches := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		if h.Data != nil {
			matches = append(matches, h.Data)
			continue
		}
		matches = append(matches, map[string]any{
			"session_id": h.ID, "score": h.Score, "snippet": h.Snippet,
		})
	}
	return map[string]any{"count": len(hits), "matches": matches}
}

// SessionRanker reorders session recall hits (e.g. via cognition Ranker).
type SessionRanker func(ctx context.Context, hits []RecallHit) ([]RecallHit, error)

type Port interface {
	Recall(ctx context.Context, q RecallQuery) (RecallResult, error)
	Store(ctx context.Context, rec Record) error
	Compress(ctx context.Context, in CompressInput) (CompressOutput, error)
}

// noopPort is the default no-op implementation.
type noopPort struct{}

// Noop returns a Port that leaves messages unchanged and ignores recall/store.
func Noop() Port { return noopPort{} }

func (noopPort) Recall(context.Context, RecallQuery) (RecallResult, error) {
	return RecallResult{}, nil
}

func (noopPort) Store(context.Context, Record) error { return nil }

func (noopPort) Compress(_ context.Context, in CompressInput) (CompressOutput, error) {
	tok := in.EstimatedTokens
	if tok <= 0 {
		tok = len(in.Messages)
	}
	return CompressOutput{
		Messages:             in.Messages,
		PreviousSummary:      in.PreviousSummary,
		EstimatedTokensAfter: tok,
	}, nil
}
