package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// renderTable lays out rows under headers with no borders and no trailing spaces.
func renderTable(st styles, headers []string, rows [][]string) string {
	t := table.New().
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderHeader(false).BorderColumn(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := st.cell
			if row == table.HeaderRow {
				s = st.header
			}
			if col < len(headers)-1 {
				s = s.PaddingRight(columnGap)
			}
			return s
		})
	return trimLines(t.Render())
}

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}
