package cmd

import (
	"errors"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/store"
)

var errNoDatabase = errors.New("no database yet")

// openExisting opens the database without creating it, or returns errNoDatabase.
func openExisting(cmd *cobra.Command) (*store.Store, error) {
	path, err := dbPath(cmd)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, errNoDatabase
	}
	return store.Open(cmd.Context(), path)
}
