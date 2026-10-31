package comparison

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/compare"
	"github.com/gian-gg/metron/internal/render/internal/ui"
	"github.com/gian-gg/metron/internal/store/db"
)

func comparisonFixture() []compare.Metric {
	return []compare.Metric{
		{Name: "traces", Unit: compare.UnitCount, A: 2, B: 1, SampleSize: true},
		{Name: "duration (mean)", Unit: compare.UnitDuration, A: float64(2 * time.Second), B: float64(1500 * time.Millisecond), LowerIsBetter: true},
		{Name: "traces with errors", Unit: compare.UnitPercent, A: 50, B: 0, LowerIsBetter: true},
		{Name: "model calls / trace", Unit: compare.UnitCount, A: 1.5, B: 1.5},
		{Name: "tool calls / trace", Unit: compare.UnitCount, A: 0, B: 2},
		{Name: "input tokens / trace", Unit: compare.UnitCount, A: 200, B: 300, LowerIsBetter: true},
	}
}

func TestRender(t *testing.T) {
	left := Side{Tag: "v1", Sessions: []db.Session{{ID: "aaaaaaaaaaaa"}, {ID: "bbbbbbbbbbbb"}}}
	right := Side{Sessions: []db.Session{{ID: "cccccccccccc"}}}
	var buf bytes.Buffer

	if err := Render(&buf, left, right, comparisonFixture()); err != nil {
		t.Fatalf("render: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(lines[0], "v1 (2 sessions)") || !strings.Contains(lines[0], "cccccccc (1 session)") {
		t.Errorf("intro line should name both sides: %q", lines[0])
	}
	table := strings.Join(lines[2:], "\n")
	for _, want := range []string{"METRIC", "v1", "cccccccc", "CHANGE"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("header missing %q: %s", want, lines[2])
		}
	}
	rows := map[string]string{}
	for _, line := range lines[3:] {
		rows[strings.Fields(line)[0]+" "+strings.Fields(line)[1]] = line
	}
	wantRows := map[string][]string{
		"traces 2":        {"2", "1", ui.Placeholder},
		"duration (mean)": {"2s", "1.5s", "-25%"},
		"traces with":     {"50%", "0%", "-100%"},
		"model calls":     {"1.5", "1.5", "0%"},
		"tool calls":      {"0", "2", ui.Placeholder},
		"input tokens":    {"200", "300", "+50%"},
	}
	for key, wants := range wantRows {
		line, ok := rows[key]
		if !ok {
			t.Errorf("row %q missing in:\n%s", key, table)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(line, want) {
				t.Errorf("row %q missing %q: %s", key, want, line)
			}
		}
	}
}

func TestRenderIsPlainWhenPiped(t *testing.T) {
	side := Side{Tag: "v1", Sessions: []db.Session{{ID: "aaaaaaaaaaaa"}}}
	var buf bytes.Buffer

	if err := Render(&buf, side, side, comparisonFixture()); err != nil {
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

func TestNumber(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "0"}, {2, "2"}, {1.5, "1.5"}, {2.25, "2.2"}, {23908.4, "23908"}, {100, "100"},
	}
	for _, tt := range tests {
		if got := number(tt.v); got != tt.want {
			t.Errorf("number(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestChangeText(t *testing.T) {
	tests := []struct {
		name string
		m    compare.Metric
		want string
	}{
		{"drop", compare.Metric{A: 200, B: 50}, "-75%"},
		{"rise", compare.Metric{A: 100, B: 150}, "+50%"},
		{"same", compare.Metric{A: 10, B: 10}, "0%"},
		{"tiny change", compare.Metric{A: 1000, B: 1001}, "~0%"},
		{"from zero", compare.Metric{A: 0, B: 5}, ui.Placeholder},
		{"sample size", compare.Metric{A: 1, B: 3, SampleSize: true}, ui.Placeholder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := changeText(tt.m); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
