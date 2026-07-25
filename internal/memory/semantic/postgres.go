package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
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
func NewPostgresStore(db *sql.DB, cfg *config.AppConfig) *PostgresStore {
	return &PostgresStore{db: db, embedder: NewEmbedderFromConfig(cfg)}
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

// AddFact stores a distilled semantic fact (multiple per session allowed).
func (s *PostgresStore) AddFact(ctx context.Context, sessionID, userID, subject, content string) error {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" || content == "" {
		return nil
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("semantic store not configured")
	}
	ctx = contextOrBackground(ctx)
	text := fmt.Sprintf("[%s] %s", subject, content)
	var vec any
	if s.embedder != nil {
		if emb, err := s.embedder.Embed(ctx, text); err == nil && len(emb) > 0 {
			vec = vectorLiteral(emb)
		}
	}
	if vec != nil {
		_, err := s.db.ExecContext(ctx, `
            INSERT INTO agent_memory_chunks (session_id, user_id, source, content, embedding, created_at)
            VALUES ($1,$2,'fact',$3,$4::vector,NOW())`,
			sessionID, userID, text, vec)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO agent_memory_chunks (session_id, user_id, source, content, created_at)
        VALUES ($1,$2,'fact',$3,NOW())`,
		sessionID, userID, text)
	return err
}

// Create stores a manual semantic chunk and returns its id.
func (s *PostgresStore) Create(ctx context.Context, sessionID, userID, source, content string) (int64, error) {
	content = strings.TrimSpace(content)
	source = strings.TrimSpace(source)
	if content == "" {
		return 0, fmt.Errorf("content required")
	}
	if source == "" {
		source = "manual"
	}
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("semantic store not configured")
	}
	ctx = contextOrBackground(ctx)
	var vec any
	if s.embedder != nil {
		if emb, err := s.embedder.Embed(ctx, content); err == nil && len(emb) > 0 {
			vec = vectorLiteral(emb)
		}
	}
	var id int64
	var err error
	if vec != nil {
		err = s.db.QueryRowContext(ctx, `
            INSERT INTO agent_memory_chunks (session_id, user_id, source, content, embedding, created_at)
            VALUES ($1,$2,$3,$4,$5::vector,NOW()) RETURNING id`,
			sessionID, userID, source, content, vec).Scan(&id)
	} else {
		err = s.db.QueryRowContext(ctx, `
            INSERT INTO agent_memory_chunks (session_id, user_id, source, content, created_at)
            VALUES ($1,$2,$3,$4,NOW()) RETURNING id`,
			sessionID, userID, source, content).Scan(&id)
	}
	return id, err
}

// GetByID returns one chunk row.
func (s *PostgresStore) GetByID(ctx context.Context, id int64) (*Chunk, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("semantic store not configured")
	}
	var c Chunk
	err := s.db.QueryRowContext(ctx, `
        SELECT id, session_id, user_id, source, content, created_at
        FROM agent_memory_chunks WHERE id=$1`, id).Scan(
		&c.ID, &c.SessionID, &c.UserID, &c.Source, &c.Content, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateContent replaces chunk text (clears embedding for simplicity).
func (s *PostgresStore) UpdateContent(ctx context.Context, id int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("content required")
	}
	if s == nil || s.db == nil {
		return fmt.Errorf("semantic store not configured")
	}
	res, err := s.db.ExecContext(ctx, `
        UPDATE agent_memory_chunks SET content=$2, embedding=NULL WHERE id=$1`, id, content)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a chunk row.
func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("semantic store not configured")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_memory_chunks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
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

// RecallChunks finds nearest semantic chunks, optionally scoped to a user.
func (s *PostgresStore) RecallChunks(ctx context.Context, query, userID, excludeSessionID string, limit int) ([]Chunk, error) {
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
          AND ($2 = '' OR user_id = $2)
          AND ($3 = '' OR session_id <> $3)
        ORDER BY embedding <=> $1::vector
        LIMIT $4`, vectorLiteral(emb), strings.TrimSpace(userID), strings.TrimSpace(excludeSessionID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChunks(rows)
}

// SearchVector finds nearest chunks when embeddings exist.
func (s *PostgresStore) SearchVector(ctx context.Context, query string, limit int) ([]Chunk, error) {
	return s.RecallChunks(ctx, query, "", "", limit)
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
