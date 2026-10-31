// Package session holds what the read commands share: finding and opening the database.
package session

import (
	"errors"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/store"
)

// NoSessionsHint is printed when there is nothing to show yet.
const NoSessionsHint = "No sessions yet. Start recording with: metron serve"

// ErrNoDatabase means no database exists yet.
var ErrNoDatabase = errors.New("no database yet")

// DBPath is the database path given by the --db flag.
func DBPath(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("db")
}

// OpenExisting opens the database without creating it, or returns ErrNoDatabase.
func OpenExisting(cmd *cobra.Command) (*store.Store, error) {
	path, err := DBPath(cmd)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoDatabase
	}
	return store.Open(cmd.Context(), path)
}
