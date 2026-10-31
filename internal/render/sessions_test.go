package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

func TestStatus(t *testing.T) {
	end := int64(1)
	tests := []struct {
		name    string
		session db.Session
		want    string
	}{
		{"running", db.Session{}, "running"},
		{"done", db.Session{EndTime: &end}, "done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status(tt.session); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionsTable(t *testing.T) {
	now := time.Unix(10_000, 0)
	start := now.Add(-2 * time.Minute).UnixNano()
	end := start + (5100 * time.Millisecond).Nanoseconds()
	tag := "v2"

	sessions := []db.Session{{ID: "a1b2c3d4e5f6", Tag: &tag, StartTime: start, EndTime: &end}}

	var buf bytes.Buffer
	if err := Sessions(&buf, sessions, now); err != nil {
		t.Fatalf("render: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want header + 1 row:\n%s", len(lines), buf.String())
	}
	for _, want := range []string{"ID", "TAG", "STARTED", "DURATION", "STATUS"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header missing %q: %s", want, lines[0])
		}
	}
	for _, want := range []string{"a1b2c3d4", "v2", "2m ago", "5.1s", "done"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row missing %q: %s", want, lines[1])
		}
	}
	if strings.Contains(lines[1], "e5f6") {
		t.Errorf("id not shortened: %s", lines[1])
	}
}

func TestSessionsTableEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	if err := Sessions(&buf, []db.Session{{ID: "x"}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := strings.Count(buf.String(), "—"); got != 2 {
		t.Errorf("got %d placeholders, want 2 (tag, duration):\n%s", got, buf.String())
	}
}

func TestSessionsTableHasNoColorWhenPiped(t *testing.T) {
	end := int64(1)
	var buf bytes.Buffer
	if err := Sessions(&buf, []db.Session{{ID: "x", EndTime: &end}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output contains color codes:\n%q", buf.String())
	}
}

func TestStatusCellIcons(t *testing.T) {
	end := int64(1)
	st := newStyles(&bytes.Buffer{})
	tests := []struct {
		name    string
		session db.Session
		want    string
	}{
		{"running", db.Session{}, "● running"},
		{"done", db.Session{EndTime: &end}, "✓ done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCell(tt.session, st); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionsTableHasNoTrailingSpaces(t *testing.T) {
	end := int64(1)
	var buf bytes.Buffer
	if err := Sessions(&buf, []db.Session{{ID: "x", EndTime: &end}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("trailing space: %q", line)
		}
	}
}
