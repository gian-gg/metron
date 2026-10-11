package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

const (
	defaultLimit = 20
	noRunsHint   = "No runs yet. Record one with: metron run -- <command>"
)

func newLsCmd() *cobra.Command {
	var (
		limit int
		tag   string
	)

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List recent runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := dbPath(cmd)
			if err != nil {
				return err
			}

			if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noRunsHint)
				return nil
			}

			s, err := store.Open(cmd.Context(), path)
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			params := db.ListRunsParams{Limit: int64(limit)}
			if tag != "" {
				params.Tag = &tag
			}

			runs, err := s.ListRuns(cmd.Context(), params)
			if err != nil {
				return fmt.Errorf("list runs: %w", err)
			}

			if len(runs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), noRunsHint)
				return nil
			}
			return render.Runs(cmd.OutOrStdout(), runs, time.Now())
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultLimit, "maximum number of runs to show")
	cmd.Flags().StringVar(&tag, "tag", "", "only show runs with this tag")
	return cmd
}
