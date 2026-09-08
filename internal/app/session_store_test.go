package app_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestSessionStoreFallbackToSQLiteWhenPostgresUnavailable(t *testing.T) {
	t.Setenv("GEEGOO_SESSION_STORE", "postgres")
	t.Setenv("GEEGOO_PG_DSN", "postgres://invalid:5432/nope?connect_timeout=1&sslmode=disable")

	root := t.TempDir()
	db, err := infra.OpenSQLite(filepath.Join(root, "geegoo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	application := &app.App{
		DB:    db,
		State: infra.NewStateStore(root),
	}
	store, err := application.SessionStore()
	if err != nil {
		t.Fatalf("SessionStore() err=%v", err)
	}
	if _, ok := store.(*chatsession.SQLiteSessionStore); !ok {
		t.Fatalf("expected sqlite fallback store, got %T", store)
	}
	if _, err := store.Create(); err != nil {
		t.Fatalf("Create() err=%v", err)
	}
}
