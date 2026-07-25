package episodic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory/fts"
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

// Format formats an episode like Waku search results.
func Format(happenedAt time.Time, summary string) string {
	return fmt.Sprintf("(%s) %s", happenedAt.Format("2006-01-02"), strings.TrimSpace(summary))
}

// Add stores a new episode.
func (s *PostgresStore) Add(ctx context.Context, sessionID, userID, summary string, happenedAt time.Time) error {
	if strings.TrimSpace(summary) == "" {
		return nil
	}
	_, err := s.Create(ctx, sessionID, userID, summary, happenedAt)
	return err
}

// Create stores a new episode and returns its id.
func (s *PostgresStore) Create(ctx context.Context, sessionID, userID, summary string, happenedAt time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("episodic store not configured")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return 0, fmt.Errorf("summary required")
	}
	if happenedAt.IsZero() {
		happenedAt = time.Now().UTC()
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `
        INSERT INTO agent_episodes (session_id, user_id, happened_at, summary, created_at)
        VALUES ($1,$2,$3,$4,NOW()) RETURNING id`,
		sessionID, userID, happenedAt.Format("2006-01-02"), summary).Scan(&id)
	return id, err
}

// GetByID returns one episode row.
func (s *PostgresStore) GetByID(ctx context.Context, id int64) (*Episode, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("episodic store not configured")
	}
	var e Episode
	var happenedAt time.Time
	err := s.db.QueryRowContext(ctx, `
        SELECT id, session_id, user_id, happened_at, summary, created_at
        FROM agent_episodes WHERE id=$1`, id).Scan(
		&e.ID, &e.SessionID, &e.UserID, &happenedAt, &e.Summary, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	e.HappenedAt = happenedAt
	return &e, nil
}

// Update replaces summary and happened_at for an episode.
func (s *PostgresStore) Update(ctx context.Context, id int64, summary string, happenedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("episodic store not configured")
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("summary required")
	}
	if happenedAt.IsZero() {
		happenedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
        UPDATE agent_episodes SET summary=$2, happened_at=$3 WHERE id=$1`,
		id, summary, happenedAt.Format("2006-01-02"))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes an episode row.
func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("episodic store not configured")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_episodes WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SearchEpisodes finds episodes matching query text, scoped by user.
func (s *PostgresStore) SearchEpisodes(ctx context.Context, query, userID string, limit int) ([]Episode, error) {
	return s.Search(ctx, query, userID, limit)
}

// Search finds episodes via FTS when query is set (Waku episodes_fts parity).
func (s *PostgresStore) Search(ctx context.Context, query, userID string, limit int) ([]Episode, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	userID = strings.TrimSpace(userID)
	query = strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	ftsQ := fts.BuildQuery(query)
	if ftsQ == "" {
		rows, err = s.db.QueryContext(ctx, `
            SELECT id, session_id, user_id, happened_at, summary, created_at
            FROM agent_episodes
            WHERE ($1 = '' OR user_id = $1)
            ORDER BY happened_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
            SELECT id, session_id, user_id, happened_at, summary, created_at
            FROM agent_episodes
            WHERE ($1 = '' OR user_id = $1)
              AND search_vector @@ to_tsquery('simple', $2)
            ORDER BY ts_rank(search_vector, to_tsquery('simple', $2)) DESC,
                     happened_at DESC, id DESC
            LIMIT $3`, userID, ftsQ, limit)
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
