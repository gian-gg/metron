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

	"github.com/gian-gg/metron/internal/store/db"
)

const (
	listenAddr        = "127.0.0.1:0"
	readHeaderTimeout = 10 * time.Second
)

// SpanWriter saves converted spans.
type SpanWriter interface {
	InsertSpans(ctx context.Context, spans []db.InsertSpanParams) error
}

// Receiver accepts OTLP trace exports for one run on a single local port.
type Receiver struct {
	runID  string
	writer SpanWriter
	server *http.Server
	grpc   *grpc.Server
	mux    *http.ServeMux
	addr   string
	count  atomic.Int64
}

// Start listens on a random local port and serves OTLP over HTTP and gRPC.
func Start(runID string, w SpanWriter) (*Receiver, error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for spans: %w", err)
	}

	r := &Receiver{
		runID:  runID,
		writer: w,
		grpc:   grpc.NewServer(),
		mux:    http.NewServeMux(),
		addr:   ln.Addr().String(),
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

	go func() {
		if err := r.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = ln.Close()
		}
	}()
	return r, nil
}

// Endpoint is the base URL to give OTLP exporters.
func (r *Receiver) Endpoint() string {
	return "http://" + r.addr
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
	rows, err := toRows(r.runID, req)
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
