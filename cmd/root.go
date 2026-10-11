package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/store"
)

// Version is the metron version, set at build time.
var Version = "dev"

// NewRootCmd builds the metron command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "metron",
		Short:         "Local-first observability for AI agents",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().String("db", store.DefaultPath, "path to the metron database")

	root.AddCommand(newLsCmd(), newRunCmd())
	return root
}

func dbPath(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("db")
}
