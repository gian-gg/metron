package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/compare"
	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store"
)

func newCompareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compare <a> <b>",
		Short: "Compare two tags or sessions side by side",
		Long: "Compare two tags or sessions side by side, averaging over their traces.\n" +
			"Each argument is a tag, which covers every session recorded under it, or a session ID or ID prefix.\n" +
			"The change column is b relative to a.",
		Args: cobra.ExactArgs(2),
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

			left, leftTraces, err := loadSide(cmd.Context(), s, args[0])
			if err != nil {
				return err
			}
			right, rightTraces, err := loadSide(cmd.Context(), s, args[1])
			if err != nil {
				return err
			}
			return render.Comparison(cmd.OutOrStdout(), left, right, compare.Metrics(leftTraces, rightTraces))
		},
	}
}

// loadSide resolves name to a group and returns its traces, failing when it has none.
func loadSide(ctx context.Context, s *store.Store, name string) (render.Side, []compare.Trace, error) {
	group, err := s.FindGroup(ctx, name)
	if err != nil {
		return render.Side{}, nil, err
	}
	stats, err := s.GroupTraceStats(ctx, group)
	if err != nil {
		return render.Side{}, nil, err
	}
	if len(stats) == 0 {
		return render.Side{}, nil, fmt.Errorf("no traces recorded under %q", name)
	}
	return render.Side{Tag: group.Tag, Sessions: group.Sessions}, compare.FromStats(stats), nil
}
