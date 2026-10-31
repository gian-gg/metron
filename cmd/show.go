package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [session]",
		Short: "Show the spans of a session as a timeline",
		Long:  "Show the spans of a session as a timeline, grouped by trace.\nThe session is an ID or ID prefix from ls; the latest session is used when none is given.",
		Args:  cobra.MaximumNArgs(1),
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

			session, found, err := pickSession(cmd.Context(), s, args)
			if err != nil {
				return err
			}
			if !found {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noSessionsHint)
				return nil
			}

			spans, err := s.ListSessionSpans(cmd.Context(), session.ID)
			if err != nil {
				return fmt.Errorf("list spans: %w", err)
			}
			return render.Timeline(cmd.OutOrStdout(), session, spans)
		},
	}
}

// pickSession returns the session named by args, or the latest one when none is named.
func pickSession(ctx context.Context, s *store.Store, args []string) (db.Session, bool, error) {
	if len(args) == 1 {
		session, err := s.FindSession(ctx, args[0])
		return session, err == nil, err
	}
	latest, err := s.ListSessions(ctx, db.ListSessionsParams{Limit: 1})
	if err != nil {
		return db.Session{}, false, fmt.Errorf("list sessions: %w", err)
	}
	if len(latest) == 0 {
		return db.Session{}, false, nil
	}
	return latest[0], true, nil
}
