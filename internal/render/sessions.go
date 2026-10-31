package render

import (
	"fmt"
	"io"
	"time"

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

	_, err := fmt.Fprintln(w, renderTable(st, sessionHeaders, rows))
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
	return runningCell(status(s) == statusRunning, st)
}

func runningCell(running bool, st styles) string {
	if running {
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
