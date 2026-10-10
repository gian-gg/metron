package store_test

import (
	"context"
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

	runs, err := s.ListRuns(ctx, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != "new" || runs[1].ID != "old" {
		t.Fatalf("got %+v, want new then old", runs)
	}
}
