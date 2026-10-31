package ui

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

// Styles are the text styles used for terminal output.
type Styles struct {
	Header  lipgloss.Style
	Cell    lipgloss.Style
	Dim     lipgloss.Style
	OK      lipgloss.Style
	Fail    lipgloss.Style
	Running lipgloss.Style
}

// NewStyles returns styles that emit color only when w is a color terminal.
func NewStyles(w io.Writer) Styles {
	r := lipgloss.NewRenderer(w)
	gray := lipgloss.Color("243")
	return Styles{
		Header:  r.NewStyle().Bold(true).Foreground(gray),
		Cell:    r.NewStyle(),
		Dim:     r.NewStyle().Foreground(gray),
		OK:      r.NewStyle().Foreground(lipgloss.Color("2")),
		Fail:    r.NewStyle().Foreground(lipgloss.Color("1")),
		Running: r.NewStyle().Foreground(lipgloss.Color("3")),
	}
}
