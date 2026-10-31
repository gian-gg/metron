package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

func TestShowWithoutDatabaseShowsHintAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metron", "metron.db")

	out := execute(t, "show", "--db", path)

	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("missing hint:\n%s", out)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("show created %s", filepath.Dir(path))
	}
}

func TestShowEmptyDatabaseShowsHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path)

	out := execute(t, "show", "--db", path)

	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("missing hint:\n%s", out)
	}
}

func TestShowDefaultsToTheLatestSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", StartTime: 2},
	)
	seedSpans(t, path, span("aaaaaaaaaaaa", "s1", "old-span", "other", 10), span("bbbbbbbbbbbb", "s2", "new-span", "tool", 20))

	out := execute(t, "show", "--db", path)

	if !strings.Contains(out, "session bbbbbbbb") || !strings.Contains(out, "new-span") || strings.Contains(out, "old-span") {
		t.Errorf("want only the latest session:\n%s", out)
	}
}

func TestShowSelectsASessionByPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", StartTime: 2},
	)
	seedSpans(t, path, span("aaaaaaaaaaaa", "s1", "old-span", "other", 10), span("bbbbbbbbbbbb", "s2", "new-span", "tool", 20))

	out := execute(t, "show", "aaaa", "--db", path)

	if !strings.Contains(out, "session aaaaaaaa") || !strings.Contains(out, "old-span") || strings.Contains(out, "new-span") {
		t.Errorf("want only session aaaa:\n%s", out)
	}
}

func TestShowRejectsUnknownAndAmbiguousSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path,
		db.CreateSessionParams{ID: "abc111111111", StartTime: 1},
		db.CreateSessionParams{ID: "abc222222222", StartTime: 2},
	)
	tests := []struct {
		arg     string
		wantErr error
	}{
		{"zzz", store.ErrSessionNotFound},
		{"abc", store.ErrSessionAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			_, err := executeErr(t, "show", tt.arg, "--db", path)

			if err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) || !strings.Contains(err.Error(), tt.arg) {
				t.Errorf("err = %v, want one containing %q and %q", err, tt.wantErr, tt.arg)
			}
		})
	}
}

func TestShowSessionWithoutSpans(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path, db.CreateSessionParams{ID: "aaaaaaaaaaaa", StartTime: 1})

	out := execute(t, "show", "--db", path)

	if !strings.Contains(out, "No spans yet") {
		t.Errorf("missing empty hint:\n%s", out)
	}
}
