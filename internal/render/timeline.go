package render

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

const (
	kindOther   = "other"
	statusError = "error"
)

var timelineHeaders = []string{"OFFSET", "KIND", "NAME", "MODEL", "DURATION", "TOKENS", "STATUS"}

type trace struct {
	id    string
	spans []db.ListSessionSpansRow
}

// Timeline writes a header for tag (or for the one session when tag is empty), then the spans grouped by trace and ordered by start time.
func Timeline(w io.Writer, tag string, sessions []db.Session, spans []db.ListSessionSpansRow) error {
	st := newStyles(w)
	if _, err := fmt.Fprintln(w, timelineHeader(tag, sessions, st)); err != nil {
		return err
	}
	if len(spans) == 0 {
		_, err := fmt.Fprintln(w, st.dim.Render("No spans yet."))
		return err
	}

	for _, tr := range groupTraces(spans) {
		if _, err := fmt.Fprintf(w, "\n%s\n", traceHeader(tr, st)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, renderTable(st, timelineHeaders, timelineRows(tr, st))); err != nil {
			return err
		}
	}
	return nil
}

func timelineHeader(tag string, sessions []db.Session, st styles) string {
	if tag == "" {
		return sessionHeader(sessions[0], st)
	}
	running := slices.ContainsFunc(sessions, func(s db.Session) bool { return s.EndTime == nil })
	state := statusDone
	if running {
		state = statusRunning
	}
	return strings.Join([]string{"tag " + tag, sessionCount(len(sessions)), state}, st.dim.Render(" · "))
}

func sessionHeader(s db.Session, st styles) string {
	sep := st.dim.Render(" · ")
	parts := []string{"session " + shortID(s.ID)}
	if s.Tag != nil && *s.Tag != "" {
		parts = append(parts, "tag "+*s.Tag)
	}
	parts = append(parts, status(s), sessionDuration(s))
	return strings.Join(parts, sep)
}

func traceHeader(tr trace, st styles) string {
	start, end, errs := traceBounds(tr)
	parts := []string{"trace " + shortID(tr.id), duration(time.Duration(end - start)), spanCount(len(tr.spans))}
	if errs > 0 {
		parts = append(parts, st.fail.Render(errorCount(errs)))
	}
	return strings.Join(parts, st.dim.Render(" · "))
}

func timelineRows(tr trace, st styles) [][]string {
	start, _, _ := traceBounds(tr)
	rows := make([][]string, 0, len(tr.spans))
	for _, sp := range tr.spans {
		rows = append(rows, []string{
			st.dim.Render("+" + duration(time.Duration(sp.StartTime-start))),
			kindCell(sp.Kind, st),
			sp.Name,
			dimIfPlaceholder(orPlaceholder(sp.Model), st),
			duration(time.Duration(sp.EndTime - sp.StartTime)),
			dimIfPlaceholder(tokens(sp.InputTokens, sp.OutputTokens), st),
			statusMark(sp.Status, st),
		})
	}
	return rows
}

func kindCell(kind string, st styles) string {
	if kind == kindOther {
		return st.dim.Render(kind)
	}
	return kind
}

func statusMark(status string, st styles) string {
	if status == statusError {
		return st.fail.Render("✗ error")
	}
	return ""
}

// groupTraces splits spans into traces, in the order each trace first appears.
func groupTraces(spans []db.ListSessionSpansRow) []trace {
	var traces []trace
	index := map[string]int{}
	for _, sp := range spans {
		i, ok := index[sp.TraceID]
		if !ok {
			i = len(traces)
			index[sp.TraceID] = i
			traces = append(traces, trace{id: sp.TraceID})
		}
		traces[i].spans = append(traces[i].spans, sp)
	}
	return traces
}

func traceBounds(tr trace) (start, end int64, errs int) {
	start, end = tr.spans[0].StartTime, tr.spans[0].EndTime
	for _, sp := range tr.spans {
		start, end = min(start, sp.StartTime), max(end, sp.EndTime)
		if sp.Status == statusError {
			errs++
		}
	}
	return start, end, errs
}

// tokens formats token counts, hiding absent and zero values.
func tokens(in, out *int64) string {
	var parts []string
	if in != nil && *in > 0 {
		parts = append(parts, fmt.Sprintf("%d in", *in))
	}
	if out != nil && *out > 0 {
		parts = append(parts, fmt.Sprintf("%d out", *out))
	}
	if len(parts) == 0 {
		return placeholder
	}
	return strings.Join(parts, " · ")
}

func sessionCount(n int) string {
	if n == 1 {
		return "1 session"
	}
	return fmt.Sprintf("%d sessions", n)
}

func errorCount(n int) string {
	if n == 1 {
		return "1 error"
	}
	return fmt.Sprintf("%d errors", n)
}
