package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

const (
	defaultLimit   = 20
	noSessionsHint = "No sessions yet. Start recording with: metron serve"
)

type lsOptions struct {
	limit    int
	tag      string
	sessions bool
}

func newLsCmd() *cobra.Command {
	var opts lsOptions

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List recorded sessions, grouped by tag",
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

			return list(cmd, s, opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "n", defaultLimit, "maximum number of rows to show")
	cmd.Flags().StringVar(&opts.tag, "tag", "", "only show sessions with this tag")
	cmd.Flags().BoolVar(&opts.sessions, "sessions", false, "list every session instead of grouping by tag")
	return cmd
}

func list(cmd *cobra.Command, s *store.Store, opts lsOptions) error {
	var tag *string
	if opts.tag != "" {
		tag = &opts.tag
	}
	out := cmd.OutOrStdout()

	if opts.sessions {
		sessions, err := s.ListSessions(cmd.Context(), db.ListSessionsParams{Tag: tag, Limit: int64(opts.limit)})
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		if len(sessions) == 0 {
			_, _ = fmt.Fprintln(out, noSessionsHint)
			return nil
		}
		return render.Sessions(out, sessions, time.Now())
	}

	groups, err := s.ListGroups(cmd.Context(), db.ListGroupsParams{Tag: tag, Limit: int64(opts.limit)})
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}
	if len(groups) == 0 {
		_, _ = fmt.Fprintln(out, noSessionsHint)
		return nil
	}
	return render.Groups(out, groups, time.Now())
}
