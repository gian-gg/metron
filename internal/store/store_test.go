package store_test

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

func openTemp(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".metron", "metron.db")
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestOpenCreatesDatabaseFile(t *testing.T) {
	_, path := openTemp(t)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file not created: %v", err)
	}
}

func TestOpenAppliesSettings(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	tests := []struct {
		pragma string
		want   string
	}{
		{"journal_mode", "wal"},
		{"foreign_keys", "1"},
		{"busy_timeout", "5000"},
	}
	for _, tt := range tests {
		t.Run(tt.pragma, func(t *testing.T) {
			var got string
			if err := s.DB.QueryRowContext(ctx, "PRAGMA "+tt.pragma).Scan(&got); err != nil {
				t.Fatalf("read pragma: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	ctx := context.Background()

	for range 2 {
		s, err := store.Open(ctx, path)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if err := s.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}
}

func TestListSessionsNewestFirst(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	for i, id := range []string{"old", "new"} {
		err := s.CreateSession(ctx, db.CreateSessionParams{ID: id, StartTime: int64(i)})
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	sessions, err := s.ListSessions(ctx, db.ListSessionsParams{Limit: 10})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 || sessions[0].ID != "new" || sessions[1].ID != "old" {
		t.Fatalf("got %+v, want new then old", sessions)
	}
}

func TestListSessionsFiltersByTag(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	v1, v2 := "v1", "v2"
	for _, r := range []db.CreateSessionParams{
		{ID: "a", Tag: &v1},
		{ID: "b", Tag: &v2},
		{ID: "c"},
	} {
		if err := s.CreateSession(ctx, r); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	sessions, err := s.ListSessions(ctx, db.ListSessionsParams{Tag: &v2, Limit: 10})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "b" {
		t.Fatalf("got %+v, want only session b", sessions)
	}
}

func TestNewSessionIDIsShortUniqueHex(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := store.NewSessionID()
		if len(id) != 12 {
			t.Fatalf("id %q has length %d, want 12", id, len(id))
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Fatalf("id %q is not hex", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func spanRow(sessionID, spanID string) db.InsertSpanParams {
	return db.InsertSpanParams{
		TraceID: "t1", SpanID: spanID, SessionID: sessionID, Name: "chat",
		Status: "ok", StartTime: 1, EndTime: 2, Attributes: "{}",
	}
}

func countSpans(t *testing.T, s *store.Store) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM spans").Scan(&n); err != nil {
		t.Fatalf("count spans: %v", err)
	}
	return n
}

func TestInsertSpans(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	sessionID := "session1"
	if err := s.CreateSession(ctx, db.CreateSessionParams{ID: sessionID}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	batch := []db.InsertSpanParams{spanRow(sessionID, "s1"), spanRow(sessionID, "s2")}

	if err := s.InsertSpans(ctx, batch); err != nil {
		t.Fatalf("insert spans: %v", err)
	}
	if err := s.InsertSpans(ctx, batch); err != nil {
		t.Fatalf("insert duplicate batch: %v", err)
	}

	if got := countSpans(t, s); got != 2 {
		t.Errorf("got %d spans, want 2 (duplicates ignored)", got)
	}
}

func TestInsertSpansRejectsUnknownSession(t *testing.T) {
	s, _ := openTemp(t)
	err := s.InsertSpans(context.Background(), []db.InsertSpanParams{spanRow("missing", "s1")})

	if err == nil {
		t.Fatal("want foreign key error for unknown session")
	}
	if got := countSpans(t, s); got != 0 {
		t.Errorf("got %d spans, want 0 after failed batch", got)
	}
}
