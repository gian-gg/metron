package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/gian-gg/metron/internal/store/db"
)

var runHeaders = []string{"ID", "TAG", "CASE", "STARTED", "DURATION", "STATUS"}

// Runs writes runs as a table, with times relative to now.
func Runs(w io.Writer, runs []db.Run, now time.Time) error {
	st := newStyles(w)

	rows := make([][]string, 0, len(runs))
	for _, r := range runs {
		rows = append(rows, []string{
			shortID(r.ID),
			dimIfPlaceholder(orPlaceholder(r.Tag), st),
			dimIfPlaceholder(orPlaceholder(r.CaseName), st),
			st.dim.Render(ago(now.Sub(time.Unix(0, r.StartTime)))),
			dimIfPlaceholder(runDuration(r), st),
			statusCell(r, st),
		})
	}

	t := table.New().
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).
		Headers(runHeaders...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := st.cell
			if row == table.HeaderRow {
				s = st.header
			}
			if col < len(runHeaders)-1 {
				s = s.PaddingRight(columnGap)
			}
			return s
		})

	_, err := fmt.Fprintln(w, trimLines(t.Render()))
	return err
}

// Recorded writes a one-line summary of a finished run and how many spans it sent.
func Recorded(w io.Writer, r db.Run, spans int) error {
	st := newStyles(w)
	sep := st.dim.Render(" · ")
	_, err := fmt.Fprintln(w,
		st.dim.Render("● recorded "+shortID(r.ID))+sep+statusCell(r, st)+sep+runDuration(r)+sep+spanCount(spans),
	)
	return err
}

func spanCount(n int) string {
	if n == 1 {
		return "1 span"
	}
	return fmt.Sprintf("%d spans", n)
}

func runDuration(r db.Run) string {
	if r.EndTime == nil {
		return placeholder
	}
	return duration(time.Duration(*r.EndTime - r.StartTime))
}

func status(r db.Run) string {
	switch {
	case r.EndTime == nil || r.ExitCode == nil:
		return "running"
	case *r.ExitCode == 0:
		return "ok"
	default:
		return fmt.Sprintf("exit %d", *r.ExitCode)
	}
}

func statusCell(r db.Run, st styles) string {
	s := status(r)
	switch s {
	case "running":
		return st.running.Render("● " + s)
	case "ok":
		return st.ok.Render("✓ " + s)
	default:
		return st.fail.Render("✗ " + s)
	}
}

func dimIfPlaceholder(s string, st styles) string {
	if s == placeholder {
		return st.dim.Render(s)
	}
	return s
}

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}
