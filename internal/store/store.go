package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver

	"github.com/gian-gg/metron/internal/store/db"
)

// DefaultPath is where metron keeps its database, relative to the working directory.
const DefaultPath = ".metron/metron.db"

const runIDBytes = 6

const pragmas = "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

//go:embed migrations/*.sql
var migrations embed.FS

// Store is an open metron database with typed queries.
type Store struct {
	DB *sql.DB
	*db.Queries
}

// Open creates the database at path if needed, applies settings and runs migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}

	conn, err := sql.Open("sqlite", "file:"+path+pragmas)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Store{DB: conn, Queries: db.New(conn)}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.DB.Close()
}

func migrate(ctx context.Context, conn *sql.DB) error {
	dir, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, dir)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// InsertSpans saves a batch of spans in one transaction, skipping spans already stored.
func (s *Store) InsertSpans(ctx context.Context, spans []db.InsertSpanParams) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin span batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := s.WithTx(tx)
	for _, span := range spans {
		if err := q.InsertSpan(ctx, span); err != nil {
			return fmt.Errorf("insert span %s: %w", span.SpanID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit span batch: %w", err)
	}
	return nil
}

// NewRunID returns a random 12-character hex run ID.
func NewRunID() string {
	b := make([]byte, runIDBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
