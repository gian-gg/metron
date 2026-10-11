package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gian-gg/metron/internal/store/db"
)

func TestLsWithoutDatabaseShowsHintAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metron", "metron.db")

	out := execute(t, "ls", "--db", path)

	if !strings.Contains(out, "No runs yet") {
		t.Errorf("missing hint:\n%s", out)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("ls created %s", filepath.Dir(path))
	}
}

func TestLsListsRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path, db.CreateRunParams{ID: "run-one-id", Command: "agent", StartTime: 1})

	out := execute(t, "ls", "--db", path)

	if !strings.Contains(out, "run-one-") {
		t.Errorf("run missing:\n%s", out)
	}
}

func TestLsFiltersByTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	v1, v2 := "v1", "v2"
	seed(t, path,
		db.CreateRunParams{ID: "aaaaaaaa", Tag: &v1, Command: "agent", StartTime: 1},
		db.CreateRunParams{ID: "bbbbbbbb", Tag: &v2, Command: "agent", StartTime: 2},
	)

	out := execute(t, "ls", "--db", path, "--tag", "v2")

	if strings.Contains(out, "aaaaaaaa") || !strings.Contains(out, "bbbbbbbb") {
		t.Errorf("want only v2 run:\n%s", out)
	}
}

func TestLsEmptyDatabaseShowsHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seed(t, path)

	out := execute(t, "ls", "--db", path)

	if !strings.Contains(out, "No runs yet") {
		t.Errorf("missing hint:\n%s", out)
	}
}
