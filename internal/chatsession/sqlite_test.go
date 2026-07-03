package chatsession_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func newSQLiteStore(t *testing.T) *chatsession.SQLiteSessionStore {
	t.Helper()
	db, err := infra.OpenSQLite(filepath.Join(t.TempDir(), "geegoo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return chatsession.NewSQLiteSessionStore(db)
}

func TestSQLiteSessionCreateSaveLoad(t *testing.T) {
	t.Parallel()
	store := newSQLiteStore(t)
	sess, err := store.Create()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.ID == "" || sess.Status != "active" {
		t.Fatalf("bad session: %+v", sess)
	}
	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: "查腾讯股价"})
	sess.RefreshMetadata()
	if err := store.Save(sess); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil || loaded.ID != sess.ID {
		t.Fatalf("loaded mismatch: %+v", loaded)
	}
	if len(loaded.Messages) != 2 || loaded.Messages[1].Content != "查腾讯股价" {
		t.Fatalf("messages lost: %+v", loaded.Messages)
	}
	if loaded.Title != "查腾讯股价" {
		t.Fatalf("title not derived: %q", loaded.Title)
	}
}

func TestSQLiteSessionListAndIDs(t *testing.T) {
	t.Parallel()
	store := newSQLiteStore(t)
	for i := 0; i < 3; i++ {
		sess, _ := store.Create()
		sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: "q"})
		_ = store.Save(sess)
	}
	ids, err := store.ListSessionIDs()
	if err != nil {
		t.Fatalf("list ids: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	entries, err := store.ListIndexedSessions()
	if err != nil {
		t.Fatalf("list indexed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestSQLiteSessionFTS(t *testing.T) {
	t.Parallel()
	store := newSQLiteStore(t)
	sess, _ := store.Create()
	sess.Messages = append(sess.Messages, llm.Message{Role: llm.RoleUser, Content: "SpaceX 上市了吗"})
	sess.RefreshMetadata()
	_ = store.Save(sess)

	ids, err := store.SearchFTS("SpaceX", 5)
	if err != nil {
		t.Fatalf("fts: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("FTS returned no hits for SpaceX")
	}
	if ids[0] != sess.ID {
		t.Fatalf("FTS hit id mismatch: %s", ids[0])
	}
}
