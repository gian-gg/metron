package ls_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gian-gg/metron/internal/store/db"
	"github.com/gian-gg/metron/internal/testutil"
)

func TestLsWithoutDatabaseShowsHintAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metron", "metron.db")

	out := testutil.Execute(t, "ls", "--db", path)

	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("missing hint:\n%s", out)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("ls created %s", filepath.Dir(path))
	}
}

func TestLsListsSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	testutil.Seed(t, path, db.CreateSessionParams{ID: "session-one-id", StartTime: 1})

	out := testutil.Execute(t, "ls", "--db", path)

	if !strings.Contains(out, "session-") {
		t.Errorf("session missing:\n%s", out)
	}
}

func TestLsGroupsSessionsSharingATag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1 := "v1"
	testutil.Seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
		db.CreateSessionParams{ID: "cccccccccccc", StartTime: 3},
	)

	out := testutil.Execute(t, "ls", "--db", path)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want header + the untagged session + the v1 group:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[1], "cccccccc") || !strings.Contains(lines[2], "v1") || !strings.Contains(lines[2], "2") {
		t.Errorf("want untagged row then a v1 row of 2 sessions:\n%s", out)
	}
	if strings.Contains(out, "aaaaaaaa") || strings.Contains(out, "bbbbbbbb") {
		t.Errorf("grouped view should not list the tagged sessions by id:\n%s", out)
	}
}

func TestLsSessionsFlagListsEverySession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1 := "v1"
	testutil.Seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
	)

	out := testutil.Execute(t, "ls", "--sessions", "--db", path)

	if !strings.Contains(out, "aaaaaaaa") || !strings.Contains(out, "bbbbbbbb") {
		t.Errorf("want both sessions listed by id:\n%s", out)
	}
}

func TestLsFiltersByTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1, v2 := "v1", "v2"
	testutil.Seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbb", Tag: &v2, StartTime: 2},
	)

	grouped := testutil.Execute(t, "ls", "--db", path, "--tag", "v2")
	sessions := testutil.Execute(t, "ls", "--sessions", "--db", path, "--tag", "v2")

	if strings.Contains(grouped, "v1") || !strings.Contains(grouped, "v2") {
		t.Errorf("want only the v2 group:\n%s", grouped)
	}
	if strings.Contains(sessions, "aaaaaaaa") || !strings.Contains(sessions, "bbbbbbbb") {
		t.Errorf("want only the v2 session:\n%s", sessions)
	}
}

func TestLsEmptyDatabaseShowsHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	testutil.Seed(t, path)

	out := testutil.Execute(t, "ls", "--db", path)

	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("missing hint:\n%s", out)
	}
}
