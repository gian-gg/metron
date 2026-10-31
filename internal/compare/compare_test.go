package compare_test

import (
	"math"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/compare"
	"github.com/gian-gg/metron/internal/store/db"
)

func byName(metrics []compare.Metric) map[string]compare.Metric {
	out := map[string]compare.Metric{}
	for _, m := range metrics {
		out[m.Name] = m
	}
	return out
}

func TestMetrics(t *testing.T) {
	a := []compare.Trace{
		{Duration: time.Second, ModelCalls: 1, InputTokens: 100, OutputTokens: 10},
		{Duration: 3 * time.Second, Errors: 2, ModelCalls: 3, ToolCalls: 2, InputTokens: 300, OutputTokens: 30},
	}
	b := []compare.Trace{
		{Duration: 2 * time.Second, ModelCalls: 2, ToolCalls: 1, InputTokens: 50, OutputTokens: 5},
	}

	got := byName(compare.Metrics(a, b))

	tests := []struct {
		name         string
		wantA, wantB float64
		unit         compare.Unit
		lowerBetter  bool
	}{
		{"traces", 2, 1, compare.UnitCount, false},
		{"duration (mean)", float64(2 * time.Second), float64(2 * time.Second), compare.UnitDuration, true},
		{"duration (median)", float64(2 * time.Second), float64(2 * time.Second), compare.UnitDuration, true},
		{"traces with errors", 50, 0, compare.UnitPercent, true},
		{"model calls / trace", 2, 2, compare.UnitCount, false},
		{"tool calls / trace", 1, 1, compare.UnitCount, false},
		{"input tokens / trace", 200, 50, compare.UnitCount, true},
		{"output tokens / trace", 20, 5, compare.UnitCount, true},
	}
	if !got["traces"].SampleSize || got["duration (mean)"].SampleSize {
		t.Errorf("only the trace count is a sample size: %+v", got)
	}
	if len(got) != len(tests) {
		t.Fatalf("got %d metrics, want %d: %v", len(got), len(tests), got)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := got[tt.name]
			if !ok {
				t.Fatalf("metric missing")
			}
			if m.A != tt.wantA || m.B != tt.wantB || m.Unit != tt.unit || m.LowerIsBetter != tt.lowerBetter {
				t.Errorf("got %+v, want A=%v B=%v unit=%v lowerIsBetter=%v", m, tt.wantA, tt.wantB, tt.unit, tt.lowerBetter)
			}
		})
	}
}

func TestMetricsKeepRowOrder(t *testing.T) {
	one := []compare.Trace{{Duration: time.Second}}

	metrics := compare.Metrics(one, one)

	if len(metrics) == 0 || metrics[0].Name != "traces" || metrics[len(metrics)-1].Name != "output tokens / trace" {
		t.Errorf("rows out of order: %+v", metrics)
	}
}

func TestMedianDurationOfAnOddCount(t *testing.T) {
	traces := []compare.Trace{{Duration: time.Second}, {Duration: 5 * time.Second}, {Duration: 2 * time.Second}}

	got := byName(compare.Metrics(traces, traces))["duration (median)"]

	if got.A != float64(2*time.Second) {
		t.Errorf("median = %v, want 2s", time.Duration(got.A))
	}
}

func TestMetricsNeedTracesOnBothSides(t *testing.T) {
	one := []compare.Trace{{Duration: time.Second}}

	if got := compare.Metrics(nil, one); got != nil {
		t.Errorf("empty left side gave %v, want nil", got)
	}
	if got := compare.Metrics(one, nil); got != nil {
		t.Errorf("empty right side gave %v, want nil", got)
	}
}

func TestChange(t *testing.T) {
	tests := []struct {
		name   string
		a, b   float64
		want   float64
		wantOK bool
	}{
		{"drop", 200, 50, -75, true},
		{"rise", 100, 150, 50, true},
		{"same", 10, 10, 0, true},
		{"from zero has no percentage", 0, 5, 0, false},
		{"both zero", 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := compare.Metric{A: tt.a, B: tt.b}.Change()

			if ok != tt.wantOK || math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestFromStats(t *testing.T) {
	rows := []db.ListSessionTraceStatsRow{{
		TraceID: "t1", Started: 1_000, Ended: 3_000_001_000, Errors: 2, ModelCalls: 3, ToolCalls: 4, InputTokens: 500, OutputTokens: 50,
	}}

	got := compare.FromStats(rows)

	want := compare.Trace{Duration: 3 * time.Second, Errors: 2, ModelCalls: 3, ToolCalls: 4, InputTokens: 500, OutputTokens: 50}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v, want [%+v]", got, want)
	}
}
