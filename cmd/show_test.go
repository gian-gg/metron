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

func TestShowByTagMergesEverySessionWithIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1 := "v1"
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
		db.CreateSessionParams{ID: "cccccccccccc", StartTime: 3},
	)
	seedSpans(t, path,
		span("aaaaaaaaaaaa", "s1", "first-span", "other", 10),
		span("bbbbbbbbbbbb", "s2", "second-span", "tool", 20),
		span("cccccccccccc", "s3", "untagged-span", "tool", 30),
	)

	out := execute(t, "show", "v1", "--db", path)

	if !strings.Contains(out, "tag v1") || !strings.Contains(out, "2 sessions") {
		t.Errorf("header should name the tag and its sessions:\n%s", out)
	}
	if !strings.Contains(out, "first-span") || !strings.Contains(out, "second-span") || strings.Contains(out, "untagged-span") {
		t.Errorf("want the spans of both v1 sessions and nothing else:\n%s", out)
	}
}

func TestShowByIDOfATaggedSessionShowsOnlyThatSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1 := "v1"
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
	)
	seedSpans(t, path, span("aaaaaaaaaaaa", "s1", "first-span", "other", 10), span("bbbbbbbbbbbb", "s2", "second-span", "tool", 20))

	out := execute(t, "show", "aaaa", "--db", path)

	if !strings.Contains(out, "session aaaaaaaa") || !strings.Contains(out, "first-span") || strings.Contains(out, "second-span") {
		t.Errorf("want only session aaaa:\n%s", out)
	}
}

func TestShowWithoutAnArgumentUsesTheLatestGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1 := "v1"
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", StartTime: 2},
		db.CreateSessionParams{ID: "cccccccccccc", Tag: &v1, StartTime: 3},
	)
	seedSpans(t, path,
		span("aaaaaaaaaaaa", "s1", "first-span", "other", 10),
		span("bbbbbbbbbbbb", "s2", "middle-span", "tool", 20),
		span("cccccccccccc", "s3", "last-span", "tool", 30),
	)

	out := execute(t, "show", "--db", path)

	if !strings.Contains(out, "tag v1") || !strings.Contains(out, "first-span") || !strings.Contains(out, "last-span") || strings.Contains(out, "middle-span") {
		t.Errorf("want the whole v1 group:\n%s", out)
	}
}
