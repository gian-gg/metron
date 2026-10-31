package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store/db"
)

const (
	defaultLimit   = 20
	noSessionsHint = "No sessions yet. Start recording with: metron serve"
)

func newLsCmd() *cobra.Command {
	var (
		limit int
		tag   string
	)

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List recorded sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := openExisting(cmd)
			if errors.Is(err, errNoDatabase) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noSessionsHint)
				return nil
			}
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			params := db.ListSessionsParams{Limit: int64(limit)}
			if tag != "" {
				params.Tag = &tag
			}

			sessions, err := s.ListSessions(cmd.Context(), params)
			if err != nil {
				return fmt.Errorf("list sessions: %w", err)
			}

			if len(sessions) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noSessionsHint)
				return nil
			}
			return render.Sessions(cmd.OutOrStdout(), sessions, time.Now())
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultLimit, "maximum number of sessions to show")
	cmd.Flags().StringVar(&tag, "tag", "", "only show sessions with this tag")
	return cmd
}
