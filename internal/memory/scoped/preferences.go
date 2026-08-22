package scoped

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

// PreferencesStore persists scoped preference text (replaces primary AGENTS.md storage).
type PreferencesStore struct {
	db *sql.DB
}

// NewPreferencesStore creates a store when PostgreSQL is configured.
func NewPreferencesStore(db *sql.DB) *PreferencesStore {
	if db == nil {
		return nil
	}
	return &PreferencesStore{db: db}
}

// Get returns preference markdown for a scope.
func (s *PreferencesStore) Get(ctx context.Context, userID, scope string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, nil
	}
	scope = NormalizeScope(scope)
	userID = strings.TrimSpace(userID)
	var content string
	err := s.db.QueryRowContext(ctx, `
        SELECT content FROM agent_scoped_preferences
        WHERE user_id = $1 AND scope = $2`, userID, scope).Scan(&content)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", false, nil
	}
	return content, true, nil
}

// Put upserts preference content for a scope.
func (s *PreferencesStore) Put(ctx context.Context, userID, scope, content, source string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("scoped preferences store not configured")
	}
	scope = NormalizeScope(scope)
	content = strings.TrimSpace(content)
	if content == "" {
		return chatprompt.ErrAgentsEmpty()
	}
	if len([]byte(content)) > chatprompt.AgentsMaxBytes {
		return chatprompt.ErrAgentsTooLarge()
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "ops"
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO agent_scoped_preferences (user_id, scope, content, source, updated_at)
        VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (user_id, scope) DO UPDATE
        SET content = EXCLUDED.content, source = EXCLUDED.source, updated_at = NOW()`,
		strings.TrimSpace(userID), scope, strings.TrimRight(content, "\n")+"\n", source)
	return err
}

// AppendRule adds one bullet line to a scope preference (chat tools).
func (s *PreferencesStore) AppendRule(ctx context.Context, userID, scope, rule, source string) error {
	rule = strings.TrimSpace(strings.TrimLeft(rule, "-"))
	if rule == "" {
		return nil
	}
	existing, ok, err := s.Get(ctx, userID, scope)
	if err != nil {
		return err
	}
	line := "- " + rule
	if !ok || strings.TrimSpace(existing) == "" {
		return s.Put(ctx, userID, scope, line+"\n", source)
	}
	text := strings.TrimRight(existing, "\n") + "\n" + line + "\n"
	return s.Put(ctx, userID, scope, text, source)
}

// CountLoaded returns scopes with non-empty content for a user.
func (s *PreferencesStore) CountLoaded(ctx context.Context, userID string) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM agent_scoped_preferences
        WHERE user_id = $1 AND content <> ''`, strings.TrimSpace(userID)).Scan(&n)
	return n, err
}

// ProfileBackend implements chatprompt.ProfileBackend (DB first, file fallback).
type ProfileBackend struct {
	Home string
	DB   *PreferencesStore
}

func (b *ProfileBackend) Get(ctx context.Context, userID string, ref chatprompt.ProfileRef) (string, bool) {
	if b == nil {
		return "", false
	}
	scope := ScopeFromRef(ref)
	if b.DB != nil {
		if c, ok, err := b.DB.Get(ctx, userID, scope); err == nil && ok {
			return c, true
		}
	}
	if lp, ok := chatprompt.LoadProfile(b.Home, userID, ref); ok {
		return lp.Content, true
	}
	return "", false
}

func (b *ProfileBackend) Put(ctx context.Context, userID string, ref chatprompt.ProfileRef, content string) error {
	if b == nil {
		return fmt.Errorf("profile backend not configured")
	}
	scope := ScopeFromRef(ref)
	if b.DB != nil {
		return b.DB.Put(ctx, userID, scope, content, "ops")
	}
	return chatprompt.SaveProfile(b.Home, userID, ref, content)
}
