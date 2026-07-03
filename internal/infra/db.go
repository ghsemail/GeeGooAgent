package infra

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// dsnParams are appended to the SQLite file path as query parameters.
// modernc.org/sqlite honors `_pragma` repeated params to set pragmas at open time.
const dsnParams = "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)&_pragma=journal_mode(wal)"

//go:embed schema.sql
var schemaFS embed.FS

// DB wraps a SQLite connection with WAL + foreign keys enabled.
type DB struct {
	sql    *sql.DB
	mu     sync.Mutex
	closed bool
}

// OpenSQLite opens or creates a SQLite database at path with pragmatic PRAGMAs
// and runs idempotent schema migrations.
func OpenSQLite(path string) (*DB, error) {
	dsn := buildDSN(path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	conn.SetMaxOpenConns(1) // SQLite write serialized; WAL allows concurrent readers
	if err := conn.PingContext(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{sql: conn}
	if err := db.applySchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func buildDSN(path string) string {
	return path + dsnParams
}

func (d *DB) applySchema() error {
	raw, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read embedded schema: %w", err)
	}
	// Execute statements one by one; modernc supports multi-statement but splitting
	// gives clearer error attribution.
	for _, stmt := range splitSQLStatements(string(raw)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := d.sql.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("apply schema stmt [%.80s...]: %w", stmt, err)
		}
	}
	return nil
}

func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return d.sql.Close()
}

// SQL exposes the underlying *sql.DB for store implementations.
func (d *DB) SQL() *sql.DB { return d.sql }

// Ping verifies connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.sql.PingContext(ctx)
}

func splitSQLStatements(script string) []string {
	// Naive splitter that respects statement boundaries at line-start "CREATE"/"PRAGMA"/"CREATE INDEX".
	// schema.sql is hand-maintained and simple; this is sufficient and avoids a full SQL parser.
	var out []string
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if (strings.HasPrefix(upper, "CREATE ") || strings.HasPrefix(upper, "PRAGMA ")) && cur.Len() > 0 {
			flush()
		}
		cur.WriteString(line)
		cur.WriteString("\n")
	}
	flush()
	return out
}

// NowRFC3339 returns current UTC time in RFC3339, the canonical timestamp format for this store.
func NowRFC3339() string { return time.Now().UTC().Format(time.RFC3339Nano) }
