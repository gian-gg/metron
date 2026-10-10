// Command metron records and inspects AI agent runs.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gian-gg/metron/cmd"
)

func main() {
	if err := cmd.NewRootCmd().ExecuteContext(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "metron:", err)
		os.Exit(1)
	}
}
