package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/internal/receiver"
	"github.com/gian-gg/metron/internal/render"
	"github.com/gian-gg/metron/internal/runner"
	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

const (
	exitCannotStart         = 127
	receiverShutdownTimeout = 5 * time.Second
)

type runLabels struct {
	tag      string
	caseName string
}

func newRunCmd() *cobra.Command {
	var labels runLabels

	cmd := &cobra.Command{
		Use:   "run [flags] -- <command> [args...]",
		Short: "Run a command and record it",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer signal.Stop(signals)

			return record(cmd, args, labels, signals)
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().StringVar(&labels.tag, "tag", "", "label for the version being run, used by compare")
	cmd.Flags().StringVar(&labels.caseName, "case", "", "label for the input being run")
	return cmd
}

func record(cmd *cobra.Command, args []string, labels runLabels, signals <-chan os.Signal) error {
	ctx := cmd.Context()

	path, err := dbPath(cmd)
	if err != nil {
		return err
	}
	s, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	run, err := startRecord(ctx, s, args, labels)
	if err != nil {
		return err
	}

	rcv, err := receiver.Start(run.ID, s)
	if err != nil {
		if _, finishErr := finishRecord(ctx, s, run, exitCannotStart); finishErr != nil {
			return finishErr
		}
		return &ExitError{Code: exitCannotStart, Err: err}
	}

	code, runErr := runAgent(cmd, args, rcv.Endpoint(), signals)
	stopReceiver(ctx, rcv)

	run, err = finishRecord(ctx, s, run, code)
	if err != nil {
		return err
	}

	if runErr != nil {
		return &ExitError{Code: code, Err: runErr}
	}
	if err := render.Recorded(cmd.ErrOrStderr(), run, rcv.SpanCount()); err != nil {
		return err
	}
	if code != 0 {
		return &ExitError{Code: code}
	}
	return nil
}

func runAgent(cmd *cobra.Command, args []string, endpoint string, signals <-chan os.Signal) (int, error) {
	code, err := runner.Run(cmd.Context(), runner.Config{
		Args:    args,
		Env:     receiver.Env(os.Environ(), endpoint),
		Stdin:   cmd.InOrStdin(),
		Stdout:  cmd.OutOrStdout(),
		Stderr:  cmd.ErrOrStderr(),
		Signals: signals,
	})
	if err != nil {
		return exitCannotStart, err
	}
	return code, nil
}

func stopReceiver(ctx context.Context, rcv *receiver.Receiver) {
	ctx, cancel := context.WithTimeout(ctx, receiverShutdownTimeout)
	defer cancel()
	_ = rcv.Shutdown(ctx)
}

func startRecord(ctx context.Context, s *store.Store, args []string, labels runLabels) (db.Run, error) {
	run := db.Run{
		ID:        store.NewRunID(),
		Tag:       optional(labels.tag),
		CaseName:  optional(labels.caseName),
		Command:   runner.CommandLine(args),
		StartTime: time.Now().UnixNano(),
	}
	err := s.CreateRun(ctx, db.CreateRunParams{
		ID: run.ID, Tag: run.Tag, CaseName: run.CaseName, Command: run.Command, StartTime: run.StartTime,
	})
	if err != nil {
		return db.Run{}, fmt.Errorf("create run: %w", err)
	}
	return run, nil
}

func finishRecord(ctx context.Context, s *store.Store, run db.Run, code int) (db.Run, error) {
	end, exit := time.Now().UnixNano(), int64(code)
	if err := s.FinishRun(ctx, db.FinishRunParams{EndTime: &end, ExitCode: &exit, ID: run.ID}); err != nil {
		return db.Run{}, fmt.Errorf("finish run: %w", err)
	}
	run.EndTime, run.ExitCode = &end, &exit
	return run, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
