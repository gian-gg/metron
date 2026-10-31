package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

const second = int64(time.Second)

func seedTwoTags(t *testing.T, path string) {
	t.Helper()
	v1, v2 := "v1", "v2"
	seed(t, path,
		db.CreateSessionParams{ID: "aaaaaaaaaaaa", Tag: &v1, StartTime: 1},
		db.CreateSessionParams{ID: "bbbbbbbbbbbb", Tag: &v1, StartTime: 2},
		db.CreateSessionParams{ID: "cccccccccccc", Tag: &v2, StartTime: 3},
	)
	seedSpans(t, path,
		modelSpan("aaaaaaaaaaaa", "t1", "a1", 0, 2*second, 100, 10),
		modelSpan("bbbbbbbbbbbb", "t2", "a2", 0, 4*second, 300, 30),
		modelSpan("cccccccccccc", "t3", "a3", 0, 1*second, 50, 5),
	)
}

func TestCompareTwoTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seedTwoTags(t, path)

	out := execute(t, "compare", "v1", "v2", "--db", path)

	for _, want := range []string{"v1 (2 sessions)", "v2 (1 session)", "METRIC", "CHANGE"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"traces ", "duration (mean)", "input tokens / trace"} {
		if !strings.Contains(out, "\n"+want) {
			t.Errorf("no row starting %q:\n%s", want, out)
		}
	}
	for _, want := range []string{"3s", "1s", "-67%", "200", "50", "-75%"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing value %q (mean of v1 traces is 3s and 200 input tokens, v2 is 1s and 50):\n%s", want, out)
		}
	}
}

func TestCompareAcceptsSessionIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seedTwoTags(t, path)

	out := execute(t, "compare", "aaaa", "cccc", "--db", path)

	for _, want := range []string{"aaaaaaaa (1 session)", "cccccccc (1 session)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestCompareErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	seedTwoTags(t, path)
	v3 := "v3"
	seed(t, path, db.CreateSessionParams{ID: "dddddddddddd", Tag: &v3, StartTime: 4})
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown name", []string{"compare", "v1", "nope"}, `"nope"`},
		{"side without traces", []string{"compare", "v1", "v3"}, `no traces recorded under "v3"`},
		{"one argument", []string{"compare", "v1"}, "accepts 2 arg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeErr(t, append(tt.args, "--db", path)...)

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCompareWithoutDatabaseShowsHintAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metron", "metron.db")

	out := execute(t, "compare", "v1", "v2", "--db", path)

	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("missing hint:\n%s", out)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("compare created %s", filepath.Dir(path))
	}
}
