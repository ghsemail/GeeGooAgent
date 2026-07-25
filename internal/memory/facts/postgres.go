package facts

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Row is one durable semantic fact (Waku facts table parity).
type Row struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// PostgresStore persists facts with PostgreSQL full-text search (Waku FTS5 parity).
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a facts store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Add inserts a fact. Subject is normalized to lowercase like Waku.
func (s *PostgresStore) Add(ctx context.Context, userID, subject, content, source string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("facts store not configured")
	}
	subject = normalizeSubject(subject)
	content = strings.TrimSpace(content)
	source = strings.TrimSpace(source)
	if subject == "" || content == "" {
		return nil
	}
	if source == "" {
		source = "user"
	}
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO agent_facts (user_id, subject, content, source, created_at)
        VALUES ($1,$2,$3,$4,NOW())`, strings.TrimSpace(userID), subject, content, source)
	return err
}

// Search returns formatted "[subject] content" lines for retrieval gate (Waku parity).
func (s *PostgresStore) Search(ctx context.Context, userID, query string, topK int) ([]string, error) {
	rows, err := s.SearchRows(ctx, userID, query, topK)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, Format(r.Subject, r.Content))
	}
	return out, nil
}

// SearchRows returns matching fact rows ranked by FTS.
func (s *PostgresStore) SearchRows(ctx context.Context, userID, query string, topK int) ([]Row, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if topK <= 0 {
		topK = 4
	}
	userID = strings.TrimSpace(userID)
	fts := buildFTSQuery(query)
	var (
		sqlRows *sql.Rows
		err     error
	)
	if fts == "" {
		sqlRows, err = s.db.QueryContext(ctx, `
            SELECT id, user_id, subject, content, source, created_at
            FROM agent_facts
            WHERE ($1 = '' OR user_id = $1)
            ORDER BY id DESC LIMIT $2`, userID, topK)
	} else {
		sqlRows, err = s.db.QueryContext(ctx, `
            SELECT id, user_id, subject, content, source, created_at
            FROM agent_facts
            WHERE ($1 = '' OR user_id = $1)
              AND search_vector @@ to_tsquery('simple', $2)
            ORDER BY ts_rank(search_vector, to_tsquery('simple', $2)) DESC, id DESC
            LIMIT $3`, userID, fts, topK)
	}
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()
	return scanRows(sqlRows)
}

// List returns recent facts for dashboard.
func (s *PostgresStore) List(ctx context.Context, userID string, limit int) ([]Row, error) {
	if limit <= 0 {
		limit = 200
	}
	return s.SearchRows(ctx, userID, "", limit)
}

// Update replaces content and optional subject.
func (s *PostgresStore) Update(ctx context.Context, id int64, content, subject string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("facts store not configured")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return false, fmt.Errorf("content required")
	}
	var res sql.Result
	var err error
	if strings.TrimSpace(subject) != "" {
		res, err = s.db.ExecContext(ctx,
			`UPDATE agent_facts SET content=$2, subject=$3 WHERE id=$1`,
			id, content, normalizeSubject(subject))
	} else {
		res, err = s.db.ExecContext(ctx,
			`UPDATE agent_facts SET content=$2 WHERE id=$1`, id, content)
	}
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Delete removes a fact row.
func (s *PostgresStore) Delete(ctx context.Context, id int64) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("facts store not configured")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM agent_facts WHERE id=$1`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// GetByID returns one fact.
func (s *PostgresStore) GetByID(ctx context.Context, id int64) (*Row, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("facts store not configured")
	}
	var r Row
	err := s.db.QueryRowContext(ctx, `
        SELECT id, user_id, subject, content, source, created_at
        FROM agent_facts WHERE id=$1`, id).Scan(
		&r.ID, &r.UserID, &r.Subject, &r.Content, &r.Source, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Count returns total fact rows for a user (empty userID = all).
func (s *PostgresStore) Count(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM agent_facts WHERE ($1 = '' OR user_id = $1)`,
		strings.TrimSpace(userID)).Scan(&n)
	return n, err
}

// MigrateFromLegacyChunks copies source=fact|manual rows from agent_memory_chunks.
func (s *PostgresStore) MigrateFromLegacyChunks(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT user_id, content, source FROM agent_memory_chunks
        WHERE source IN ('fact', 'manual', 'consolidation')
        ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	for rows.Next() {
		var userID, content, source string
		if err := rows.Scan(&userID, &content, &source); err != nil {
			return n, err
		}
		subject, body := splitBracketSubject(content)
		if subject == "" {
			subject = "general"
		}
		if source == "fact" {
			source = "consolidation"
		}
		if err := s.Add(ctx, userID, subject, body, source); err == nil {
			n++
		}
	}
	return n, rows.Err()
}

// Format renders a fact like Waku search results.
func Format(subject, content string) string {
	return fmt.Sprintf("[%s] %s", subject, content)
}

func normalizeSubject(subject string) string {
	return strings.ToLower(strings.TrimSpace(subject))
}

var tokenRe = regexp.MustCompile(`[a-zA-Z0-9]{2,}`)

func buildFTSQuery(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return ""
	}
	seen := map[string]struct{}{}
	var parts []string
	for _, tok := range tokenRe.FindAllString(text, -1) {
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		parts = append(parts, tok)
	}
	return strings.Join(parts, " | ")
}

func splitBracketSubject(raw string) (subject, content string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[") {
		return "", raw
	}
	end := strings.Index(raw, "]")
	if end <= 1 {
		return "", raw
	}
	return strings.TrimSpace(raw[1:end]), strings.TrimSpace(raw[end+1:])
}

func scanRows(rows *sql.Rows) ([]Row, error) {
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.UserID, &r.Subject, &r.Content, &r.Source, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
