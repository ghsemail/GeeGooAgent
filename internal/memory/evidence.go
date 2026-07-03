package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

// EvidenceStore persists EvidenceRef records and their raw payloads in SQLite.
// It is the auditable backing for working-memory EvidenceRefs: working memory
// keeps refs in-process, the store keeps the verifiable payload.
type EvidenceStore struct {
	db *infra.DB
}

// NewEvidenceStore creates a SQLite-backed evidence store.
func NewEvidenceStore(db *infra.DB) *EvidenceStore {
	return &EvidenceStore{db: db}
}

// Record persists an evidence ref plus its raw payload. Upsert by id.
// payload may be nil; in that case payload_json is stored as "{}".
func (s *EvidenceStore) Record(ref EvidenceRef, payload any) error {
	if ref.ID == "" {
		return fmt.Errorf("evidence: empty id")
	}
	payloadJSON := "{}"
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("evidence: marshal payload: %w", err)
		}
		payloadJSON = string(raw)
	}
	if ref.PayloadHash == "" {
		ref.PayloadHash = PayloadHash(payload)
	}
	ctx := context.Background()
	_, err := s.db.SQL().ExecContext(ctx, `
        INSERT INTO evidence_records
            (id, run_id, session_id, tool, source, payload_hash, summary, observed_at, payload_json)
        VALUES (?,?,?,?,?,?,?,?,?)
        ON CONFLICT(id) DO UPDATE SET
            payload_hash=excluded.payload_hash, summary=excluded.summary,
            observed_at=excluded.observed_at, payload_json=excluded.payload_json`,
		ref.ID, ref.RunID, ref.RunID, ref.Tool, ref.Source,
		ref.PayloadHash, ref.Summary, ref.ObservedAt, payloadJSON,
	)
	if err != nil {
		return fmt.Errorf("evidence: record %s: %w", ref.ID, err)
	}
	return nil
}

// Get loads a single evidence ref plus payload by id.
func (s *EvidenceStore) Get(id string) (EvidenceRef, string, error) {
	ctx := context.Background()
	var ref EvidenceRef
	var payloadJSON string
	err := s.db.SQL().QueryRowContext(ctx, `
        SELECT id, run_id, tool, source, payload_hash, summary, observed_at, payload_json
        FROM evidence_records WHERE id=?`, id,
	).Scan(&ref.ID, &ref.RunID, &ref.Tool, &ref.Source, &ref.PayloadHash,
		&ref.Summary, &ref.ObservedAt, &payloadJSON)
	if err == sql.ErrNoRows {
		return EvidenceRef{}, "", nil
	}
	if err != nil {
		return EvidenceRef{}, "", fmt.Errorf("evidence get %s: %w", id, err)
	}
	return ref, payloadJSON, nil
}

// QueryByRun returns all evidence refs for a run_id (workflow session).
func (s *EvidenceStore) QueryByRun(runID string) ([]EvidenceRef, error) {
	ctx := context.Background()
	rows, err := s.db.SQL().QueryContext(ctx, `
        SELECT id, run_id, tool, source, payload_hash, summary, observed_at
        FROM evidence_records WHERE run_id=? ORDER BY observed_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EvidenceRef
	for rows.Next() {
		var ref EvidenceRef
		if err := rows.Scan(&ref.ID, &ref.RunID, &ref.Tool, &ref.Source,
			&ref.PayloadHash, &ref.Summary, &ref.ObservedAt); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

// VerifyPayload re-hashes the stored payload and compares to payload_hash.
// Returns ok=true when the hash matches the canonical hash of payload_json.
func (s *EvidenceStore) VerifyPayload(id string) (bool, error) {
	_, payloadJSON, err := s.Get(id)
	if err != nil {
		return false, err
	}
	if payloadJSON == "" {
		return false, nil
	}
	var ref EvidenceRef
	ref, _, err = s.Get(id)
	if err != nil {
		return false, err
	}
	var payload any
	if json.Unmarshal([]byte(payloadJSON), &payload) != nil {
		payload = payloadJSON
	}
	return PayloadHash(payload) == ref.PayloadHash, nil
}

// Count returns total evidence rows.
func (s *EvidenceStore) Count() (int, error) {
	ctx := context.Background()
	var n int
	err := s.db.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM evidence_records").Scan(&n)
	return n, err
}
