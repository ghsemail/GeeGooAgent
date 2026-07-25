package episodic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Episode is one episodic memory row.
type Episode struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	HappenedAt time.Time `json:"happened_at"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

// PostgresStore persists dated episode summaries.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates an episodic store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Add stores a new episode.
func (s *PostgresStore) Add(ctx context.Context, sessionID, userID, summary string, happenedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("episodic store not configured")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}
	if happenedAt.IsZero() {
		happenedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO agent_episodes (session_id, user_id, happened_at, summary, created_at)
        VALUES ($1,$2,$3,$4,NOW())`,
		sessionID, userID, happenedAt.Format("2006-01-02"), summary)
	return err
}

// SearchEpisodes finds episodes matching query text, scoped by user.
func (s *PostgresStore) SearchEpisodes(ctx context.Context, query, userID string, limit int) ([]Episode, error) {
	return s.Search(ctx, query, userID, limit)
}

// Search finds episodes matching query text, scoped by user.
func (s *PostgresStore) Search(ctx context.Context, query, userID string, limit int) ([]Episode, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	query = strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	if query == "" {
		rows, err = s.db.QueryContext(ctx, `
            SELECT id, session_id, user_id, happened_at, summary, created_at
            FROM agent_episodes
            WHERE ($1 = '' OR user_id = $1)
            ORDER BY happened_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		pattern := "%" + query + "%"
		rows, err = s.db.QueryContext(ctx, `
            SELECT id, session_id, user_id, happened_at, summary, created_at
            FROM agent_episodes
            WHERE ($2 = '' OR user_id = $2)
              AND summary ILIKE $1
            ORDER BY happened_at DESC, id DESC LIMIT $3`, pattern, userID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEpisodes(rows)
}

// List returns recent episodes for dashboard.
func (s *PostgresStore) List(ctx context.Context, userID string, limit int) ([]Episode, error) {
	return s.Search(ctx, "", userID, limit)
}

func scanEpisodes(rows *sql.Rows) ([]Episode, error) {
	var out []Episode
	for rows.Next() {
		var e Episode
		var happenedAt time.Time
		if err := rows.Scan(&e.ID, &e.SessionID, &e.UserID, &happenedAt, &e.Summary, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.HappenedAt = happenedAt
		out = append(out, e)
	}
	return out, rows.Err()
}
