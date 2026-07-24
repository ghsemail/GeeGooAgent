package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Chunk is one semantic memory row for Cockpit / recall.
type Chunk struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// PostgresStore reads/writes agent_memory_chunks (text + optional embedding).
type PostgresStore struct {
	db       *sql.DB
	embedder Embedder
}

// NewPostgresStore creates a semantic store over PostgreSQL.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db, embedder: NewOpenAIEmbedderFromEnv()}
}

// SetEmbedder overrides the default env-based embedder.
func (s *PostgresStore) SetEmbedder(e Embedder) {
	if s != nil {
		s.embedder = e
	}
}

// UpsertSummary stores or replaces the session summary chunk (embeds when configured).
func (s *PostgresStore) UpsertSummary(ctx context.Context, sessionID, userID, summary string) error {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	ctx = contextOrBackground(ctx)
	_, err := s.db.ExecContext(ctx, `
        DELETE FROM agent_memory_chunks
        WHERE session_id=$1 AND source='session_summary'`, sessionID)
	if err != nil {
		return err
	}
	var vec any
	if s.embedder != nil {
		if emb, err := s.embedder.Embed(ctx, summary); err == nil && len(emb) > 0 {
			vec = vectorLiteral(emb)
		}
	}
	if vec != nil {
		_, err = s.db.ExecContext(ctx, `
            INSERT INTO agent_memory_chunks (session_id, user_id, source, content, embedding, created_at)
            VALUES ($1,$2,'session_summary',$3,$4::vector,NOW())`,
			sessionID, userID, summary, vec)
		return err
	}
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO agent_memory_chunks (session_id, user_id, source, content, created_at)
        VALUES ($1,$2,'session_summary',$3,NOW())`,
		sessionID, userID, summary)
	return err
}

// List returns recent chunks.
func (s *PostgresStore) List(ctx context.Context, limit int) ([]Chunk, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, session_id, user_id, source, content, created_at
        FROM agent_memory_chunks ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.SessionID, &c.UserID, &c.Source, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Count returns total chunk rows.
func (s *PostgresStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_memory_chunks`).Scan(&n)
	return n, err
}

// SearchVector finds nearest chunks when embeddings exist.
func (s *PostgresStore) SearchVector(ctx context.Context, query string, limit int) ([]Chunk, error) {
	if s == nil || s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured")
	}
	ctx = contextOrBackground(ctx)
	if limit <= 0 {
		limit = 10
	}
	emb, err := s.embedder.Embed(ctx, query)
	if err != nil || len(emb) == 0 {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, session_id, user_id, source, content, created_at
        FROM agent_memory_chunks
        WHERE embedding IS NOT NULL
        ORDER BY embedding <=> $1::vector
        LIMIT $2`, vectorLiteral(emb), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChunks(rows)
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func scanChunks(rows *sql.Rows) ([]Chunk, error) {
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.SessionID, &c.UserID, &c.Source, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SearchText does ILIKE fallback when embeddings are absent.
func (s *PostgresStore) SearchText(ctx context.Context, query string, limit int) ([]Chunk, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.List(ctx, limit)
	}
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, session_id, user_id, source, content, created_at
        FROM agent_memory_chunks
        WHERE content ILIKE $1
        ORDER BY created_at DESC LIMIT $2`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.SessionID, &c.UserID, &c.Source, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
