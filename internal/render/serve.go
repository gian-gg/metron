package render

import (
	"fmt"
	"io"
	"strings"
)

// Listening writes the line shown once the listener is up.
func Listening(w io.Writer, addrs []string, tag string) error {
	st := newStyles(w)
	line := "● listening on " + strings.Join(addrs, ", ")
	if tag != "" {
		line += " · tag " + tag
	}
	_, err := fmt.Fprintln(w, st.running.Render(line))
	return err
}

// Stopped writes the line shown when the listener has stopped.
func Stopped(w io.Writer, spans int) error {
	st := newStyles(w)
	_, err := fmt.Fprintln(w, st.dim.Render("● stopped · "+spanCount(spans)))
	return err
}

func spanCount(n int) string {
	if n == 1 {
		return "1 span"
	}
	return fmt.Sprintf("%d spans", n)
}
