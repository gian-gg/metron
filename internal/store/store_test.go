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

func TestListRunsNewestFirst(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	for i, id := range []string{"old", "new"} {
		err := s.CreateRun(ctx, db.CreateRunParams{ID: id, Command: "agent", StartTime: int64(i)})
		if err != nil {
			t.Fatalf("create run: %v", err)
		}
	}

	runs, err := s.ListRuns(ctx, db.ListRunsParams{Limit: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != "new" || runs[1].ID != "old" {
		t.Fatalf("got %+v, want new then old", runs)
	}
}

func TestListRunsFiltersByTag(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	v1, v2 := "v1", "v2"
	for _, r := range []db.CreateRunParams{
		{ID: "a", Tag: &v1, Command: "agent"},
		{ID: "b", Tag: &v2, Command: "agent"},
		{ID: "c", Command: "agent"},
	} {
		if err := s.CreateRun(ctx, r); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}

	runs, err := s.ListRuns(ctx, db.ListRunsParams{Tag: &v2, Limit: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "b" {
		t.Fatalf("got %+v, want only run b", runs)
	}
}

func TestNewRunIDIsShortUniqueHex(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := store.NewRunID()
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

func spanRow(runID *string, spanID string) db.InsertSpanParams {
	return db.InsertSpanParams{
		TraceID: "t1", SpanID: spanID, RunID: runID, Name: "chat",
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
	runID := "run1"
	if err := s.CreateRun(ctx, db.CreateRunParams{ID: runID, Command: "agent"}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	batch := []db.InsertSpanParams{spanRow(&runID, "s1"), spanRow(&runID, "s2")}

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

func TestInsertSpansRejectsUnknownRun(t *testing.T) {
	s, _ := openTemp(t)
	unknown := "missing"

	err := s.InsertSpans(context.Background(), []db.InsertSpanParams{spanRow(&unknown, "s1")})

	if err == nil {
		t.Fatal("want foreign key error for unknown run")
	}
	if got := countSpans(t, s); got != 0 {
		t.Errorf("got %d spans, want 0 after failed batch", got)
	}
}
