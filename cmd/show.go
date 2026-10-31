package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [tag or session]",
		Short: "Show the spans of a tag or session as a timeline",
		Long: "Show spans as a timeline, grouped by trace.\n" +
			"A tag shows every session recorded under it; a session ID or ID prefix from ls --sessions shows that one session.\n" +
			"With no argument, the most recent tag or session is shown.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openExisting(cmd)
			if errors.Is(err, errNoDatabase) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noSessionsHint)
				return nil
			}
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			group, found, err := pickGroup(cmd, s, args)
			if err != nil {
				return err
			}
			if !found {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noSessionsHint)
				return nil
			}

			spans, err := s.GroupSpans(cmd.Context(), group)
			if err != nil {
				return err
			}
			return render.Timeline(cmd.OutOrStdout(), group.Tag, group.Sessions, spans)
		},
	}
}

// pickGroup returns the group named by args, or the latest one when none is named.
func pickGroup(cmd *cobra.Command, s *store.Store, args []string) (store.Group, bool, error) {
	if len(args) == 1 {
		group, err := s.FindGroup(cmd.Context(), args[0])
		return group, err == nil, err
	}
	return s.LatestGroup(cmd.Context())
}
