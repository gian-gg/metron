package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
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

const sessionIDBytes = 6

// Errors returned by FindSession.
var (
	ErrSessionNotFound  = errors.New("no session matches")
	ErrSessionAmbiguous = errors.New("more than one session matches")
)

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

// NewSessionID returns a random 12-character hex session ID.
func NewSessionID() string {
	b := make([]byte, sessionIDBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// FindSession returns the one session whose ID starts with prefix.
func (s *Store) FindSession(ctx context.Context, prefix string) (db.Session, error) {
	matches, err := s.SessionsByPrefix(ctx, prefix)
	if err != nil {
		return db.Session{}, fmt.Errorf("find session: %w", err)
	}
	switch len(matches) {
	case 0:
		return db.Session{}, fmt.Errorf("%w %q", ErrSessionNotFound, prefix)
	case 1:
		return matches[0], nil
	default:
		return db.Session{}, fmt.Errorf("%w %q", ErrSessionAmbiguous, prefix)
	}
}
