package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

func TestStatus(t *testing.T) {
	zero, one, end := int64(0), int64(1), int64(1)
	tests := []struct {
		name string
		run  db.Run
		want string
	}{
		{"running", db.Run{}, "running"},
		{"ok", db.Run{EndTime: &end, ExitCode: &zero}, "ok"},
		{"failed", db.Run{EndTime: &end, ExitCode: &one}, "exit 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status(tt.run); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunsTable(t *testing.T) {
	now := time.Unix(10_000, 0)
	start := now.Add(-2 * time.Minute).UnixNano()
	end := start + (5100 * time.Millisecond).Nanoseconds()
	tag, caseName, zero := "v2", "refund-123", int64(0)

	runs := []db.Run{{
		ID: "a1b2c3d4e5f6", Tag: &tag, CaseName: &caseName,
		StartTime: start, EndTime: &end, ExitCode: &zero,
	}}

	var buf bytes.Buffer
	if err := Runs(&buf, runs, now); err != nil {
		t.Fatalf("render: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want header + 1 row:\n%s", len(lines), buf.String())
	}
	for _, want := range []string{"ID", "TAG", "CASE", "STARTED", "DURATION", "STATUS"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header missing %q: %s", want, lines[0])
		}
	}
	for _, want := range []string{"a1b2c3d4", "v2", "refund-123", "2m ago", "5.1s", "ok"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row missing %q: %s", want, lines[1])
		}
	}
	if strings.Contains(lines[1], "e5f6") {
		t.Errorf("id not shortened: %s", lines[1])
	}
}

func TestRunsTableEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	if err := Runs(&buf, []db.Run{{ID: "x"}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := strings.Count(buf.String(), "—"); got != 3 {
		t.Errorf("got %d placeholders, want 3 (tag, case, duration):\n%s", got, buf.String())
	}
}

func TestRunsTableHasNoColorWhenPiped(t *testing.T) {
	end, one := int64(1), int64(1)
	var buf bytes.Buffer
	if err := Runs(&buf, []db.Run{{ID: "x", EndTime: &end, ExitCode: &one}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output contains color codes:\n%q", buf.String())
	}
}

func TestStatusCellIcons(t *testing.T) {
	zero, one, end := int64(0), int64(1), int64(1)
	st := newStyles(&bytes.Buffer{})
	tests := []struct {
		name string
		run  db.Run
		want string
	}{
		{"running", db.Run{}, "● running"},
		{"ok", db.Run{EndTime: &end, ExitCode: &zero}, "✓ ok"},
		{"failed", db.Run{EndTime: &end, ExitCode: &one}, "✗ exit 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCell(tt.run, st); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunsTableHasNoTrailingSpaces(t *testing.T) {
	zero, end := int64(0), int64(1)
	var buf bytes.Buffer
	if err := Runs(&buf, []db.Run{{ID: "x", EndTime: &end, ExitCode: &zero}}, time.Now()); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("trailing space: %q", line)
		}
	}
}
