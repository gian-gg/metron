package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/cmd/compare"
	"github.com/gian-gg/metron/cmd/ls"
	"github.com/gian-gg/metron/cmd/serve"
	"github.com/gian-gg/metron/cmd/show"
	"github.com/gian-gg/metron/internal/store"
)

// Version is the metron version, set at build time.
var Version = "dev"

// NewRootCmd builds the metron command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "metron",
		Short:         "Local-first observability for AI agents",
		Version:       resolveVersion(Version, debug.ReadBuildInfo),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().String("db", store.DefaultPath, "path to the metron database")

	root.AddCommand(ls.New(), show.New(), compare.New(), serve.New())
	return root
}

// resolveVersion prefers the version injected at build time, then the module version of a go install.
func resolveVersion(version string, read func() (*debug.BuildInfo, bool)) string {
	if version != "dev" {
		return version
	}
	if info, ok := read(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
