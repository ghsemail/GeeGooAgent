package app

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// SessionStore returns the chat session persistence layer (SQLite preferred).
func (a *App) SessionStore() (chatsession.SessionStore, error) {
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	if a.DB != nil {
		return chatsession.NewSQLiteSessionStore(a.DB), nil
	}
	if a.State != nil {
		return chatsession.NewChatSessionStore(a.State), nil
	}
	return nil, fmt.Errorf("no session store configured")
}
