package receiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"

	"github.com/gian-gg/metron/internal/receiver/spans"
	"github.com/gian-gg/metron/internal/store/db"
)

const readHeaderTimeout = 10 * time.Second

// SpanWriter saves converted spans.
type SpanWriter interface {
	InsertSpans(ctx context.Context, spans []db.InsertSpanParams) error
}

// Receiver accepts OTLP trace exports for one session on its listen addresses.
type Receiver struct {
	sessionID string
	writer    SpanWriter
	server    *http.Server
	grpc      *grpc.Server
	mux       *http.ServeMux
	addrs     []string
	count     atomic.Int64
}

// Start listens on every address and serves OTLP over HTTP and gRPC on each.
func Start(sessionID string, w SpanWriter, addrs []string) (*Receiver, error) {
	if len(addrs) == 0 {
		return nil, errors.New("listen for spans: no address given")
	}

	listeners := make([]net.Listener, 0, len(addrs))
	for _, addr := range addrs {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			closeAll(listeners)
			return nil, fmt.Errorf("listen for spans on %s: %w", addr, err)
		}
		listeners = append(listeners, ln)
	}

	r := &Receiver{
		sessionID: sessionID,
		writer:    w,
		grpc:      grpc.NewServer(),
		mux:       http.NewServeMux(),
	}
	coltracepb.RegisterTraceServiceServer(r.grpc, &traceService{receiver: r})
	r.mux.HandleFunc("POST /v1/traces", r.handleTraces)

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	r.server = &http.Server{
		Handler:           r,
		Protocols:         &protocols,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	for _, ln := range listeners {
		r.addrs = append(r.addrs, ln.Addr().String())
		go func() {
			if err := r.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				_ = ln.Close()
			}
		}()
	}
	return r, nil
}

func closeAll(listeners []net.Listener) {
	for _, ln := range listeners {
		_ = ln.Close()
	}
}

// Addrs are the addresses the receiver is listening on.
func (r *Receiver) Addrs() []string {
	return append([]string(nil), r.addrs...)
}

// SpanCount is the number of spans accepted so far.
func (r *Receiver) SpanCount() int {
	return int(r.count.Load())
}

// Shutdown stops accepting exports and waits for in-flight ones to finish.
func (r *Receiver) Shutdown(ctx context.Context) error {
	err := r.server.Shutdown(ctx)
	r.grpc.Stop()
	return err
}

// ServeHTTP routes gRPC requests to the gRPC server and everything else to the HTTP handlers.
func (r *Receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
		r.grpc.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}

func (r *Receiver) accept(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) error {
	rows, err := spans.ToRows(r.sessionID, req)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	if err := r.writer.InsertSpans(ctx, rows); err != nil {
		return err
	}
	r.count.Add(int64(len(rows)))
	return nil
}
