package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

// SessionStore returns the chat session persistence layer.
// Priority: GEEGOO_SESSION_STORE=postgres → PostgreSQL; else SQLite; else file StateStore.
func (a *App) SessionStore() (chatsession.SessionStore, error) {
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	switch infra.SessionStoreBackend() {
	case "postgres", "pg":
		if a.PG == nil {
			return nil, fmt.Errorf("postgres session store requested but GEEGOO_PG_DSN is not connected")
		}
		return chatsession.NewPostgresSessionStore(a.PG.SQL()), nil
	case "file":
		if a.State != nil {
			return chatsession.NewChatSessionStore(a.State), nil
		}
		return nil, fmt.Errorf("file session store requested but StateStore is nil")
	case "sqlite":
		if a.DB != nil {
			return chatsession.NewSQLiteSessionStore(a.DB), nil
		}
	}
	if a.DB != nil {
		return chatsession.NewSQLiteSessionStore(a.DB), nil
	}
	if a.PG != nil && strings.TrimSpace(os.Getenv("GEEGOO_SESSION_STORE")) == "" {
		return chatsession.NewPostgresSessionStore(a.PG.SQL()), nil
	}
	if a.State != nil {
		return chatsession.NewChatSessionStore(a.State), nil
	}
	return nil, fmt.Errorf("no session store configured")
}

// SessionBackendName reports the active session persistence backend.
func (a *App) SessionBackendName() string {
	switch infra.SessionStoreBackend() {
	case "postgres", "pg":
		if a != nil && a.PG != nil {
			return "postgres"
		}
	case "file":
		return "file"
	case "sqlite":
		if a != nil && a.DB != nil {
			return "sqlite"
		}
	}
	if a != nil && a.PG != nil && infra.PostgresDSN() != "" {
		return "postgres"
	}
	if a != nil && a.DB != nil {
		return "sqlite"
	}
	return "file"
}
