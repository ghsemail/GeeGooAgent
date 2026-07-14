package infra_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestOpenSQLiteAppliesSchema(t *testing.T) {
	t.Parallel()
	db, err := infra.OpenSQLite(filepath.Join(t.TempDir(), "geegoo.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	tables := []string{
		"chat_sessions", "session_events", "evidence_records",
		"working_state", "checkpoints", "execution_events", "schema_migrations",
	}
	for _, name := range tables {
		var got string
		err := db.SQL().QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", name,
		).Scan(&got)
		if err != nil {
			t.Fatalf("missing table %s: %v", name, err)
		}
	}
	var fts string
	if err := db.SQL().QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='chat_sessions_fts'",
	).Scan(&fts); err != nil {
		t.Fatalf("missing FTS5 table: %v", err)
	}
}
