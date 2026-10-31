package ui

import (
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

// Session states.
const (
	StatusRunning = "running"
	StatusDone    = "done"
)

// SessionStatus is running until the session has an end time, then done.
func SessionStatus(s db.Session) string {
	if s.EndTime == nil {
		return StatusRunning
	}
	return StatusDone
}

// SessionDuration is how long a finished session ran, or Placeholder while it runs.
func SessionDuration(s db.Session) string {
	if s.EndTime == nil {
		return Placeholder
	}
	return Duration(time.Duration(*s.EndTime - s.StartTime))
}

// SessionStatusCell is the colored status of s.
func SessionStatusCell(s db.Session, st Styles) string {
	return RunningCell(SessionStatus(s) == StatusRunning, st)
}

// RunningCell is the colored running or done status.
func RunningCell(running bool, st Styles) string {
	if running {
		return st.Running.Render("● " + StatusRunning)
	}
	return st.OK.Render("✓ " + StatusDone)
}
