package store_test

import (
	"context"
	"encoding/hex"
	"errors"
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
		TraceID: "t1", SpanID: spanID, SessionID: sessionID, Name: "chat", Kind: "other",
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

func TestInsertSpansStoresKindModelAndTokens(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	if err := s.CreateSession(ctx, db.CreateSessionParams{ID: "session1"}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	model, in, out := "gpt-5", int64(120), int64(30)
	row := spanRow("session1", "s1")
	row.Kind, row.Model, row.InputTokens, row.OutputTokens = "model", &model, &in, &out

	if err := s.InsertSpans(ctx, []db.InsertSpanParams{row, spanRow("session1", "s2")}); err != nil {
		t.Fatalf("insert spans: %v", err)
	}

	var (
		kind              string
		gotModel          *string
		gotIn, gotOut     *int64
		plainKind         string
		plainModel        *string
		plainIn, plainOut *int64
		selectColumns     = "SELECT kind, model, input_tokens, output_tokens FROM spans WHERE span_id = ?"
	)
	if err := s.DB.QueryRowContext(ctx, selectColumns, "s1").Scan(&kind, &gotModel, &gotIn, &gotOut); err != nil {
		t.Fatalf("read s1: %v", err)
	}
	if err := s.DB.QueryRowContext(ctx, selectColumns, "s2").Scan(&plainKind, &plainModel, &plainIn, &plainOut); err != nil {
		t.Fatalf("read s2: %v", err)
	}
	if kind != "model" || gotModel == nil || *gotModel != "gpt-5" || gotIn == nil || *gotIn != 120 || gotOut == nil || *gotOut != 30 {
		t.Errorf("s1 = %q %v %v %v, want model gpt-5 120 30", kind, gotModel, gotIn, gotOut)
	}
	if plainKind != "other" || plainModel != nil || plainIn != nil || plainOut != nil {
		t.Errorf("s2 = %q %v %v %v, want other and NULLs", plainKind, plainModel, plainIn, plainOut)
	}
}

func TestSpansHaveNoCostColumn(t *testing.T) {
	s, _ := openTemp(t)

	rows, err := s.DB.QueryContext(context.Background(), "SELECT name FROM pragma_table_info('spans')")
	if err != nil {
		t.Fatalf("table info: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "cost_usd" {
			t.Fatal("spans still has cost_usd")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
}

func TestFindSession(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	for _, id := range []string{"abc123def456", "abc999000111", "fff000fff000"} {
		if err := s.CreateSession(ctx, db.CreateSessionParams{ID: id}); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	tests := []struct {
		name    string
		prefix  string
		wantID  string
		wantErr error
	}{
		{"full id", "abc123def456", "abc123def456", nil},
		{"unique prefix", "fff", "fff000fff000", nil},
		{"ambiguous prefix", "abc", "", store.ErrSessionAmbiguous},
		{"no match", "zzz", "", store.ErrSessionNotFound},
		{"wildcard is not a pattern", "%", "", store.ErrSessionNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.FindSession(ctx, tt.prefix)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got.ID != tt.wantID {
				t.Errorf("id = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestListSessionSpansOrdersByStartAndScopesToSession(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	for _, id := range []string{"s1", "s2"} {
		if err := s.CreateSession(ctx, db.CreateSessionParams{ID: id}); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
	late, early, other := spanRow("s1", "late"), spanRow("s1", "early"), spanRow("s2", "other")
	late.StartTime, early.StartTime, other.StartTime = 20, 10, 5
	if err := s.InsertSpans(ctx, []db.InsertSpanParams{late, early, other}); err != nil {
		t.Fatalf("insert spans: %v", err)
	}

	spans, err := s.ListSessionSpans(ctx, "s1")
	if err != nil {
		t.Fatalf("list spans: %v", err)
	}
	if len(spans) != 2 || spans[0].SpanID != "early" || spans[1].SpanID != "late" {
		t.Fatalf("got %+v, want early then late and nothing from s2", spans)
	}
}
