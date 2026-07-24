package infra

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/infra/pgschema"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDSN returns GEEGOO_PG_DSN when set.
func PostgresDSN() string {
	return strings.TrimSpace(os.Getenv("GEEGOO_PG_DSN"))
}

// SessionStoreBackend returns sqlite|postgres|file from GEEGOO_SESSION_STORE.
func SessionStoreBackend() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_SESSION_STORE")))
	switch v {
	case "postgres", "pg", "sqlite", "file":
		return v
	case "":
		if PostgresDSN() != "" {
			return "postgres"
		}
		return "sqlite"
	default:
		return v
	}
}

// PostgresDB is a pooled PostgreSQL connection with idempotent DDL.
type PostgresDB struct {
	sql    *sql.DB
	mu     sync.Mutex
	closed bool
}

// OpenPostgres connects and applies platform + session schemas.
func OpenPostgres(dsn string) (*PostgresDB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("empty postgres dsn")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	db := &PostgresDB{sql: conn}
	if err := db.applyCoreSchemas(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (p *PostgresDB) applyCoreSchemas() error {
	for _, name := range []string{"postgres_platform.sql", "postgres_sessions.sql"} {
		if err := p.execEmbedded(name); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresDB) execEmbedded(name string) error {
	raw, err := fs.ReadFile(pgschema.Files, name)
	if err != nil {
		return err
	}
	return p.execSQL(string(raw))
}

func (p *PostgresDB) execSQL(raw string) error {
	for _, stmt := range splitSQLStatements(raw) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := p.sql.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("postgres ddl: %w", err)
		}
	}
	return nil
}

// SQL returns the underlying database handle.
func (p *PostgresDB) SQL() *sql.DB { return p.sql }

// Close closes the pool.
func (p *PostgresDB) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.sql.Close()
}

// PingPostgres checks platform PostgreSQL reachability.
func PingPostgres(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("GEEGOO_PG_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.PingContext(ctx)
}

// ApplyMemorySchema enables pgvector tables (idempotent).
func (p *PostgresDB) ApplyMemorySchema() error {
	return p.execEmbedded("postgres_memory.sql")
}

// MemorySchemaEnabled reports whether vector tables exist.
func (p *PostgresDB) MemorySchemaEnabled() bool {
	if p == nil || p.sql == nil {
		return false
	}
	var exists bool
	err := p.sql.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'agent_memory_chunks'
		)`).Scan(&exists)
	return err == nil && exists
}
