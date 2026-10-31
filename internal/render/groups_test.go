package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

func TestGroupsTable(t *testing.T) {
	now := time.Unix(10_000, 0)
	groups := []db.ListGroupsRow{
		{
			Name: "a1b2c3d4e5f6", Tagged: 0, Sessions: 1, Traces: 1,
			Started: now.Add(-2 * time.Minute).UnixNano(), LastActive: now.Add(-time.Minute).UnixNano(),
		},
		{
			Name: "release-candidate-2", Tagged: 1, Sessions: 3, Traces: 9, Running: 1,
			Started: now.Add(-3 * time.Hour).UnixNano(), LastActive: now.Add(-10 * time.Second).UnixNano(),
		},
	}
	var buf bytes.Buffer

	if err := Groups(&buf, groups, now); err != nil {
		t.Fatalf("render: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want header + 2 rows:\n%s", len(lines), buf.String())
	}
	for _, want := range []string{"NAME", "SESSIONS", "TRACES", "STARTED", "LAST ACTIVE", "STATUS"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header missing %q: %s", want, lines[0])
		}
	}
	for _, want := range []string{"a1b2c3d4", "2m ago", "1m ago", "done"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("untagged row missing %q: %s", want, lines[1])
		}
	}
	if strings.Contains(lines[1], "e5f6") {
		t.Errorf("untagged id not shortened: %s", lines[1])
	}
	for _, want := range []string{"release-candidate-2", "3", "9", "3h ago", "10s ago", "running"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("tagged row missing %q: %s", want, lines[2])
		}
	}
}

func TestGroupsTableIsPlainWhenPiped(t *testing.T) {
	var buf bytes.Buffer

	if err := Groups(&buf, []db.ListGroupsRow{{Name: "v1", Tagged: 1, Sessions: 1}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output contains color codes:\n%q", buf.String())
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("trailing space: %q", line)
		}
	}
}
