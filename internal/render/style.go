package render

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

const columnGap = 3

type styles struct {
	header  lipgloss.Style
	cell    lipgloss.Style
	dim     lipgloss.Style
	ok      lipgloss.Style
	fail    lipgloss.Style
	running lipgloss.Style
}

// newStyles returns styles that emit color only when w is a color terminal.
func newStyles(w io.Writer) styles {
	r := lipgloss.NewRenderer(w)
	gray := lipgloss.Color("243")
	return styles{
		header:  r.NewStyle().Bold(true).Foreground(gray),
		cell:    r.NewStyle(),
		dim:     r.NewStyle().Foreground(gray),
		ok:      r.NewStyle().Foreground(lipgloss.Color("2")),
		fail:    r.NewStyle().Foreground(lipgloss.Color("1")),
		running: r.NewStyle().Foreground(lipgloss.Color("3")),
	}
}
