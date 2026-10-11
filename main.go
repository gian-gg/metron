// Command metron records and inspects AI agent runs.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gian-gg/metron/cmd"
)

func main() {
	err := cmd.NewRootCmd().ExecuteContext(context.Background())
	if err == nil {
		return
	}

	var exitErr *cmd.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "metron:", exitErr.Err)
		}
		os.Exit(exitErr.Code)
	}

	_, _ = fmt.Fprintln(os.Stderr, "metron:", err)
	os.Exit(1)
}
