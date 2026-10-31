// Package comparison renders two groups of traces side by side.
package comparison

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/gian-gg/metron/internal/compare"
	"github.com/gian-gg/metron/internal/render/internal/ui"
	"github.com/gian-gg/metron/internal/store/db"
)

// Side is one side of a comparison: every session under Tag, or one session when Tag is empty.
type Side struct {
	Tag      string
	Sessions []db.Session
}

func (s Side) label() string {
	if s.Tag != "" {
		return s.Tag
	}
	return ui.ShortID(s.Sessions[0].ID)
}

// Render writes metrics for two sides as a table, with B's change relative to A.
func Render(w io.Writer, a, b Side, metrics []compare.Metric) error {
	st := ui.NewStyles(w)
	intro := fmt.Sprintf("comparing %s (%s) with %s (%s)", a.label(), ui.SessionCount(len(a.Sessions)), b.label(), ui.SessionCount(len(b.Sessions)))
	if _, err := fmt.Fprintf(w, "%s\n\n", st.Dim.Render(intro)); err != nil {
		return err
	}

	rows := make([][]string, 0, len(metrics))
	for _, m := range metrics {
		rows = append(rows, []string{
			m.Name,
			metricValue(m.Unit, m.A),
			metricValue(m.Unit, m.B),
			changeCell(m, st),
		})
	}
	_, err := fmt.Fprintln(w, ui.Table(st, []string{"METRIC", a.label(), b.label(), "CHANGE"}, rows))
	return err
}

func metricValue(unit compare.Unit, v float64) string {
	switch unit {
	case compare.UnitDuration:
		return ui.Duration(time.Duration(v))
	case compare.UnitPercent:
		return fmt.Sprintf("%.0f%%", v)
	default:
		return number(v)
	}
}

// number prints whole numbers and large values without decimals, small fractions with one.
func number(v float64) string {
	if math.Abs(v) >= 100 || v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func changeText(m compare.Metric) string {
	pct, ok := m.Change()
	switch {
	case !ok || m.SampleSize:
		return ui.Placeholder
	case m.A == m.B:
		return "0%"
	case math.Abs(pct) < 0.5:
		return "~0%"
	default:
		return fmt.Sprintf("%+.0f%%", pct)
	}
}

// changeCell colors a change green when it moves the right way and red when it does not, for metrics where lower is better.
func changeCell(m compare.Metric, st ui.Styles) string {
	text := changeText(m)
	pct, ok := m.Change()
	if !ok || !m.LowerIsBetter || math.Abs(pct) < 0.5 {
		return text
	}
	if pct < 0 {
		return st.OK.Render(text)
	}
	return st.Fail.Render(text)
}
