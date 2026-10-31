// Package listener renders the status lines of the running listener.
package listener

import (
	"fmt"
	"io"
	"strings"

	"github.com/gian-gg/metron/internal/render/internal/ui"
)

// Listening writes the line shown once the listener is up.
func Listening(w io.Writer, addrs []string, tag string) error {
	st := ui.NewStyles(w)
	line := "● listening on " + strings.Join(addrs, ", ")
	if tag != "" {
		line += " · tag " + tag
	}
	_, err := fmt.Fprintln(w, st.Running.Render(line))
	return err
}

// Stopped writes the line shown when the listener has stopped.
func Stopped(w io.Writer, spans int) error {
	st := ui.NewStyles(w)
	_, err := fmt.Fprintln(w, st.Dim.Render("● stopped · "+ui.SpanCount(spans)))
	return err
}
