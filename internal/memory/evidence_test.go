package memory_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

func newEvidenceStore(t *testing.T) *memory.EvidenceStore {
	t.Helper()
	db, err := infra.OpenSQLite(filepath.Join(t.TempDir(), "geegoo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return memory.NewEvidenceStore(db)
}

func TestEvidenceRecordGetVerify(t *testing.T) {
	t.Parallel()
	store := newEvidenceStore(t)
	payload := map[string]any{"code": "00700.HK", "price": 312.5}
	ref := memory.NewEvidenceRef("run-1", "get_current_price", "stock.00700.HK.price",
		"腾讯 312.5", payload, time.Now())
	if err := store.Record(ref, payload); err != nil {
		t.Fatalf("record: %v", err)
	}

	loaded, payloadJSON, err := store.Get(ref.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if loaded.ID != ref.ID || loaded.Source != ref.Source {
		t.Fatalf("loaded mismatch: %+v", loaded)
	}
	if payloadJSON == "" {
		t.Fatal("payload json empty")
	}

	ok, err := store.VerifyPayload(ref.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("payload hash mismatch")
	}

	refs, err := store.QueryByRun("run-1")
	if err != nil {
		t.Fatalf("query by run: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}

	n, err := store.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected count 1, got %d", n)
	}
}

func TestEvidenceUpsertSameID(t *testing.T) {
	t.Parallel()
	store := newEvidenceStore(t)
	ref := memory.NewEvidenceRef("run-2", "get_mcp_analysis", "stock.00700.HK.weekly",
		"v1", map[string]any{"a": 1}, time.Now())
	if err := store.Record(ref, map[string]any{"a": 1}); err != nil {
		t.Fatalf("record v1: %v", err)
	}
	ref2 := ref
	ref2.Summary = "v2 updated"
	if err := store.Record(ref2, map[string]any{"a": 1}); err != nil {
		t.Fatalf("record v2: %v", err)
	}
	n, _ := store.Count()
	if n != 1 {
		t.Fatalf("expected 1 row after upsert, got %d", n)
	}
	loaded, _, _ := store.Get(ref.ID)
	if loaded.Summary != "v2 updated" {
		t.Fatalf("summary not updated: %s", loaded.Summary)
	}
}
