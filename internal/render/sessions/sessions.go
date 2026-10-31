// Package sessions renders recorded sessions as tables.
package sessions

import (
	"fmt"
	"io"
	"time"

	"github.com/gian-gg/metron/internal/render/internal/ui"
	"github.com/gian-gg/metron/internal/store/db"
)

var sessionHeaders = []string{"ID", "TAG", "STARTED", "DURATION", "STATUS"}

// List writes sessions as a table, with times relative to now.
func List(w io.Writer, sessions []db.Session, now time.Time) error {
	st := ui.NewStyles(w)

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, []string{
			ui.ShortID(s.ID),
			ui.DimIfPlaceholder(ui.OrPlaceholder(s.Tag), st),
			st.Dim.Render(ui.Ago(now.Sub(time.Unix(0, s.StartTime)))),
			ui.DimIfPlaceholder(ui.SessionDuration(s), st),
			ui.SessionStatusCell(s, st),
		})
	}

	_, err := fmt.Fprintln(w, ui.Table(st, sessionHeaders, rows))
	return err
}
