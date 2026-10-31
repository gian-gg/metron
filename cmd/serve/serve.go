// Package serve listens for OpenTelemetry traces and records them.
package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/gian-gg/metron/cmd/internal/session"
	"github.com/gian-gg/metron/internal/receiver"
	"github.com/gian-gg/metron/internal/render/listener"
	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

const (
	defaultHTTPPort         = 4318
	defaultGRPCPort         = 4317
	defaultListen           = "127.0.0.1"
	receiverShutdownTimeout = 5 * time.Second
)

type serveOptions struct {
	tag      string
	listen   string
	httpPort int
	grpcPort int
}

// New builds the serve command.
func New() *cobra.Command {
	var opts serveOptions

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Listen for OpenTelemetry traces and record them",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return serve(ctx, cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.tag, "tag", "", "label for the spans received in this session, used by compare")
	cmd.Flags().StringVar(&opts.listen, "listen", defaultListen, "address to listen on")
	cmd.Flags().IntVar(&opts.httpPort, "port", defaultHTTPPort, "port for OTLP over HTTP")
	cmd.Flags().IntVar(&opts.grpcPort, "grpc-port", defaultGRPCPort, "port for OTLP over gRPC")
	return cmd
}

func serve(ctx context.Context, cmd *cobra.Command, opts serveOptions) error {
	path, err := session.DBPath(cmd)
	if err != nil {
		return err
	}
	s, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	id := store.NewSessionID()
	err = s.CreateSession(ctx, db.CreateSessionParams{
		ID: id, Tag: optional(opts.tag), StartTime: time.Now().UnixNano(),
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	rcv, err := receiver.Start(id, s, listenAddrs(opts))
	if err != nil {
		startErr := fmt.Errorf("%w (change it with --listen, --port or --grpc-port)", err)
		if delErr := s.DeleteSession(context.WithoutCancel(ctx), id); delErr != nil {
			return errors.Join(startErr, fmt.Errorf("delete session: %w", delErr))
		}
		return startErr
	}
	if err := listener.Listening(cmd.ErrOrStderr(), rcv.Addrs(), opts.tag); err != nil {
		return err
	}

	<-ctx.Done()

	return stopServing(ctx, cmd, s, rcv, id)
}

// stopServing drains the receiver and ends the session, outliving the interrupt that stopped it.
func stopServing(ctx context.Context, cmd *cobra.Command, s *store.Store, rcv *receiver.Receiver, id string) error {
	done := context.WithoutCancel(ctx)

	shutdownCtx, cancel := context.WithTimeout(done, receiverShutdownTimeout)
	defer cancel()
	_ = rcv.Shutdown(shutdownCtx)

	end := time.Now().UnixNano()
	if err := s.FinishSession(done, db.FinishSessionParams{EndTime: &end, ID: id}); err != nil {
		return fmt.Errorf("finish session: %w", err)
	}
	return listener.Stopped(cmd.ErrOrStderr(), rcv.SpanCount())
}

func listenAddrs(opts serveOptions) []string {
	return []string{
		net.JoinHostPort(opts.listen, strconv.Itoa(opts.httpPort)),
		net.JoinHostPort(opts.listen, strconv.Itoa(opts.grpcPort)),
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
