// Package compare compares two tags or sessions side by side.
package compare

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/cmd/internal/session"
	traces "github.com/gian-gg/metron/internal/compare"
	"github.com/gian-gg/metron/internal/render/comparison"
	"github.com/gian-gg/metron/internal/store"
)

// New builds the compare command.
func New() *cobra.Command {
	return &cobra.Command{
		Use:   "compare <a> <b>",
		Short: "Compare two tags or sessions side by side",
		Long: "Compare two tags or sessions side by side, averaging over their traces.\n" +
			"Each argument is a tag, which covers every session recorded under it, or a session ID or ID prefix.\n" +
			"The change column is b relative to a.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := session.OpenExisting(cmd)
			if errors.Is(err, session.ErrNoDatabase) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), session.NoSessionsHint)
				return nil
			}
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			left, leftTraces, err := loadSide(cmd.Context(), s, args[0])
			if err != nil {
				return err
			}
			right, rightTraces, err := loadSide(cmd.Context(), s, args[1])
			if err != nil {
				return err
			}
			return comparison.Render(cmd.OutOrStdout(), left, right, traces.Metrics(leftTraces, rightTraces))
		},
	}
}

// loadSide resolves name to a group and returns its traces, failing when it has none.
func loadSide(ctx context.Context, s *store.Store, name string) (comparison.Side, []traces.Trace, error) {
	group, err := s.FindGroup(ctx, name)
	if err != nil {
		return comparison.Side{}, nil, err
	}
	stats, err := s.GroupTraceStats(ctx, group)
	if err != nil {
		return comparison.Side{}, nil, err
	}
	if len(stats) == 0 {
		return comparison.Side{}, nil, fmt.Errorf("no traces recorded under %q", name)
	}
	return comparison.Side{Tag: group.Tag, Sessions: group.Sessions}, traces.FromStats(stats), nil
}
