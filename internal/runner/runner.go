package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// signalExitBase is added to a signal number to form the exit code of a killed process.
const signalExitBase = 128

// Config describes the command to run and what it is connected to.
type Config struct {
	Args    []string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Signals <-chan os.Signal
}

// Run starts the command, forwards signals to it, and returns its exit code.
// err is only set when the command could not be started.
func Run(ctx context.Context, cfg Config) (int, error) {
	if len(cfg.Args) == 0 {
		return 0, errors.New("no command given")
	}

	c := exec.CommandContext(ctx, cfg.Args[0], cfg.Args[1:]...)
	c.Env = cfg.Env
	c.Stdin = cfg.Stdin
	c.Stdout = cfg.Stdout
	c.Stderr = cfg.Stderr

	if err := c.Start(); err != nil {
		return 0, fmt.Errorf("start %s: %w", cfg.Args[0], err)
	}

	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	for {
		select {
		case sig := <-cfg.Signals:
			if sig != os.Interrupt {
				_ = c.Process.Signal(sig)
			}
		case <-done:
			return exitCode(c.ProcessState), nil
		}
	}
}

func exitCode(state *os.ProcessState) int {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return signalExitBase + int(ws.Signal())
	}
	return state.ExitCode()
}

// CommandLine joins args into one line, quoting args that contain spaces or quotes.
func CommandLine(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'") {
			a = strconv.Quote(a)
		}
		parts[i] = a
	}
	return strings.Join(parts, " ")
}
