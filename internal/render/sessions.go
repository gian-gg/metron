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

const (
	statusRunning = "running"
	statusDone    = "done"
)

var sessionHeaders = []string{"ID", "TAG", "STARTED", "DURATION", "STATUS"}

// Sessions writes sessions as a table, with times relative to now.
func Sessions(w io.Writer, sessions []db.Session, now time.Time) error {
	st := newStyles(w)

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, []string{
			shortID(s.ID),
			dimIfPlaceholder(orPlaceholder(s.Tag), st),
			st.dim.Render(ago(now.Sub(time.Unix(0, s.StartTime)))),
			dimIfPlaceholder(sessionDuration(s), st),
			statusCell(s, st),
		})
	}

	t := table.New().
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).
		Headers(sessionHeaders...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := st.cell
			if row == table.HeaderRow {
				s = st.header
			}
			if col < len(sessionHeaders)-1 {
				s = s.PaddingRight(columnGap)
			}
			return s
		})

	_, err := fmt.Fprintln(w, trimLines(t.Render()))
	return err
}

func sessionDuration(s db.Session) string {
	if s.EndTime == nil {
		return placeholder
	}
	return duration(time.Duration(*s.EndTime - s.StartTime))
}

func status(s db.Session) string {
	if s.EndTime == nil {
		return statusRunning
	}
	return statusDone
}

func statusCell(s db.Session, st styles) string {
	if status(s) == statusRunning {
		return st.running.Render("● " + statusRunning)
	}
	return st.ok.Render("✓ " + statusDone)
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
