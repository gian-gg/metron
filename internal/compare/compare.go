package compare

import (
	"slices"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

// Unit says how a metric's values are formatted.
type Unit int

// Metric units.
const (
	UnitCount Unit = iota
	UnitDuration
	UnitPercent
)

// Trace is the totals of one agent invocation.
type Trace struct {
	Duration     time.Duration
	Errors       int
	ModelCalls   int
	ToolCalls    int
	InputTokens  int64
	OutputTokens int64
}

// Metric is one measure for both sides; durations are in nanoseconds and percentages are 0 to 100.
type Metric struct {
	Name          string
	Unit          Unit
	A, B          float64
	LowerIsBetter bool
	// SampleSize marks a count of traces, whose change is not a result.
	SampleSize bool
}

// Change is B relative to A as a percentage, or false when A is zero.
func (m Metric) Change() (float64, bool) {
	if m.A == 0 {
		return 0, false
	}
	return (m.B - m.A) / m.A * 100, true
}

// FromStats converts per-trace rows into traces.
func FromStats(rows []db.ListSessionTraceStatsRow) []Trace {
	traces := make([]Trace, 0, len(rows))
	for _, r := range rows {
		traces = append(traces, Trace{
			Duration:     time.Duration(r.Ended - r.Started),
			Errors:       int(r.Errors),
			ModelCalls:   int(r.ModelCalls),
			ToolCalls:    int(r.ToolCalls),
			InputTokens:  r.InputTokens,
			OutputTokens: r.OutputTokens,
		})
	}
	return traces
}

type measure struct {
	name          string
	unit          Unit
	lowerIsBetter bool
	sampleSize    bool
	of            func([]Trace) float64
}

var measures = []measure{
	{"traces", UnitCount, false, true, func(t []Trace) float64 { return float64(len(t)) }},
	{"duration (mean)", UnitDuration, true, false, func(t []Trace) float64 { return mean(t, func(x Trace) float64 { return float64(x.Duration) }) }},
	{"duration (median)", UnitDuration, true, false, medianDuration},
	{"traces with errors", UnitPercent, true, false, errorPercent},
	{"model calls / trace", UnitCount, false, false, func(t []Trace) float64 { return mean(t, func(x Trace) float64 { return float64(x.ModelCalls) }) }},
	{"tool calls / trace", UnitCount, false, false, func(t []Trace) float64 { return mean(t, func(x Trace) float64 { return float64(x.ToolCalls) }) }},
	{"input tokens / trace", UnitCount, true, false, func(t []Trace) float64 { return mean(t, func(x Trace) float64 { return float64(x.InputTokens) }) }},
	{"output tokens / trace", UnitCount, true, false, func(t []Trace) float64 { return mean(t, func(x Trace) float64 { return float64(x.OutputTokens) }) }},
}

// Metrics compares two sets of traces, or returns nil when either is empty.
func Metrics(a, b []Trace) []Metric {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	out := make([]Metric, 0, len(measures))
	for _, m := range measures {
		out = append(out, Metric{Name: m.name, Unit: m.unit, A: m.of(a), B: m.of(b), LowerIsBetter: m.lowerIsBetter, SampleSize: m.sampleSize})
	}
	return out
}

func mean(traces []Trace, value func(Trace) float64) float64 {
	var sum float64
	for _, t := range traces {
		sum += value(t)
	}
	return sum / float64(len(traces))
}

func medianDuration(traces []Trace) float64 {
	d := make([]float64, 0, len(traces))
	for _, t := range traces {
		d = append(d, float64(t.Duration))
	}
	slices.Sort(d)
	mid := len(d) / 2
	if len(d)%2 == 1 {
		return d[mid]
	}
	return (d[mid-1] + d[mid]) / 2
}

func errorPercent(traces []Trace) float64 {
	failed := 0
	for _, t := range traces {
		if t.Errors > 0 {
			failed++
		}
	}
	return float64(failed) / float64(len(traces)) * 100
}
