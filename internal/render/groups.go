package render

import (
	"fmt"
	"io"
	"time"

	"github.com/gian-gg/metron/internal/store/db"
)

var groupHeaders = []string{"NAME", "SESSIONS", "TRACES", "STARTED", "LAST ACTIVE", "STATUS"}

// Groups writes sessions grouped by tag as a table, with times relative to now.
func Groups(w io.Writer, groups []db.ListGroupsRow, now time.Time) error {
	st := newStyles(w)

	rows := make([][]string, 0, len(groups))
	for _, g := range groups {
		rows = append(rows, []string{
			groupName(g),
			fmt.Sprint(g.Sessions),
			fmt.Sprint(g.Traces),
			st.dim.Render(ago(now.Sub(time.Unix(0, g.Started)))),
			st.dim.Render(ago(now.Sub(time.Unix(0, g.LastActive)))),
			runningCell(g.Running != 0, st),
		})
	}

	_, err := fmt.Fprintln(w, renderTable(st, groupHeaders, rows))
	return err
}

// groupName is the tag, or the shortened ID of an untagged session.
func groupName(g db.ListGroupsRow) string {
	if g.Tagged != 0 {
		return g.Name
	}
	return shortID(g.Name)
}
