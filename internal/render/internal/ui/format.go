package ui

import (
	"fmt"
	"time"
)

const (
	shortIDLen = 8

	// Placeholder stands in for a missing value.
	Placeholder = "—"
)

// ShortID shortens an ID for display.
func ShortID(id string) string {
	if len(id) <= shortIDLen {
		return id
	}
	return id[:shortIDLen]
}

// OrPlaceholder returns s, or Placeholder when it is nil or empty.
func OrPlaceholder(s *string) string {
	if s == nil || *s == "" {
		return Placeholder
	}
	return *s
}

// DimIfPlaceholder dims s when it is Placeholder.
func DimIfPlaceholder(s string, st Styles) string {
	if s == Placeholder {
		return st.Dim.Render(s)
	}
	return s
}

// Ago formats d as a coarse "5m ago".
func Ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Duration formats d to a precision that suits its size.
func Duration(d time.Duration) string {
	switch {
	case d < time.Second:
		return d.Round(time.Millisecond).String()
	case d < time.Minute:
		return d.Round(100 * time.Millisecond).String()
	default:
		return d.Round(time.Second).String()
	}
}

// SpanCount formats a number of spans.
func SpanCount(n int) string {
	if n == 1 {
		return "1 span"
	}
	return fmt.Sprintf("%d spans", n)
}

// SessionCount formats a number of sessions.
func SessionCount(n int) string {
	if n == 1 {
		return "1 session"
	}
	return fmt.Sprintf("%d sessions", n)
}
