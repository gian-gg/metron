package receiver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gian-gg/metron/internal/store/db"
)

type fakeWriter struct {
	mu   sync.Mutex
	rows []db.InsertSpanParams
}

func (f *fakeWriter) InsertSpans(_ context.Context, spans []db.InsertSpanParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, spans...)
	return nil
}

func (f *fakeWriter) byName() map[string]db.InsertSpanParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]db.InsertSpanParams{}
	for _, r := range f.rows {
		out[r.Name] = r
	}
	return out
}

func startReceiver(t *testing.T) (*Receiver, *fakeWriter) {
	t.Helper()
	w := &fakeWriter{}
	r, err := Start("run1", w)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = r.Shutdown(context.Background()) })
	return r, w
}

// sendTrace exports an "agent" span with a failing "chat" child through exp.
func sendTrace(t *testing.T, exp *otlptrace.Exporter) {
	t.Helper()
	ctx := context.Background()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "test-agent"))),
	)
	tracer := tp.Tracer("test")

	ctx, parent := tracer.Start(ctx, "agent")
	_, child := tracer.Start(ctx, "chat")
	child.SetAttributes(attribute.String("gen_ai.request.model", "test-model"), attribute.Int("gen_ai.usage.input_tokens", 12))
	child.SetStatus(codes.Error, "boom")
	child.End()
	parent.End()

	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown tracer: %v", err)
	}
}

func assertTrace(t *testing.T, r *Receiver, w *fakeWriter) {
	t.Helper()
	spans := w.byName()
	agent, chat := spans["agent"], spans["chat"]

	if agent.SpanID == "" || chat.SpanID == "" {
		t.Fatalf("missing spans: %+v", spans)
	}
	if chat.ParentSpanID == nil || *chat.ParentSpanID != agent.SpanID {
		t.Errorf("chat parent = %v, want %s", chat.ParentSpanID, agent.SpanID)
	}
	if agent.ParentSpanID != nil {
		t.Errorf("agent parent = %v, want nil", *agent.ParentSpanID)
	}
	if chat.TraceID != agent.TraceID || len(chat.TraceID) != 32 {
		t.Errorf("trace ids %q / %q, want equal 32-char hex", chat.TraceID, agent.TraceID)
	}
	if chat.RunID == nil || *chat.RunID != "run1" {
		t.Errorf("run id = %v, want run1", chat.RunID)
	}
	if chat.Status != "error" || agent.Status != "unset" {
		t.Errorf("status chat=%q agent=%q, want error and unset", chat.Status, agent.Status)
	}
	if chat.StartTime == 0 || chat.EndTime < chat.StartTime {
		t.Errorf("bad times start=%d end=%d", chat.StartTime, chat.EndTime)
	}
	for _, want := range []string{`"gen_ai.request.model":"test-model"`, `"service.name":"test-agent"`} {
		if !strings.Contains(chat.Attributes, want) {
			t.Errorf("attributes %s missing %s", chat.Attributes, want)
		}
	}
	if got := r.SpanCount(); got != 2 {
		t.Errorf("SpanCount = %d, want 2", got)
	}
}

func TestReceiverHTTPProtobuf(t *testing.T) {
	r, w := startReceiver(t)
	exp, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpointURL(r.Endpoint()+"/v1/traces"))
	if err != nil {
		t.Fatalf("exporter: %v", err)
	}

	sendTrace(t, exp)

	assertTrace(t, r, w)
}

func TestReceiverHTTPGzip(t *testing.T) {
	r, w := startReceiver(t)
	exp, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpointURL(r.Endpoint()+"/v1/traces"),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	)
	if err != nil {
		t.Fatalf("exporter: %v", err)
	}

	sendTrace(t, exp)

	assertTrace(t, r, w)
}

func TestReceiverGRPC(t *testing.T) {
	r, w := startReceiver(t)
	exp, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithEndpointURL(r.Endpoint()))
	if err != nil {
		t.Fatalf("exporter: %v", err)
	}

	sendTrace(t, exp)

	assertTrace(t, r, w)
}

const hexJSONTrace = `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"js-agent"}}]},
"scopeSpans":[{"spans":[{"traceId":"5b8efff798038103d269b633813fc60c","spanId":"eee19b7ec3c1b174",
"parentSpanId":"eee19b7ec3c1b173","name":"chat","startTimeUnixNano":"1000","endTimeUnixNano":"2000",
"status":{"code":1}}]}]}]}`

func TestReceiverHTTPJSONUsesHexIDs(t *testing.T) {
	r, w := startReceiver(t)

	resp, err := http.Post(r.Endpoint()+"/v1/traces", "application/json", strings.NewReader(hexJSONTrace))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("response content type %q, want application/json", ct)
	}
	chat := w.byName()["chat"]
	if chat.TraceID != "5b8efff798038103d269b633813fc60c" || chat.SpanID != "eee19b7ec3c1b174" {
		t.Errorf("ids = %s/%s, want the hex values sent", chat.TraceID, chat.SpanID)
	}
	if chat.ParentSpanID == nil || *chat.ParentSpanID != "eee19b7ec3c1b173" {
		t.Errorf("parent = %v, want eee19b7ec3c1b173", chat.ParentSpanID)
	}
	if chat.Status != "ok" || chat.StartTime != 1000 || chat.EndTime != 2000 {
		t.Errorf("got status=%s start=%d end=%d", chat.Status, chat.StartTime, chat.EndTime)
	}
}

func TestReceiverRejectsUnsupportedContentType(t *testing.T) {
	r, _ := startReceiver(t)

	resp, err := http.Post(r.Endpoint()+"/v1/traces", "text/plain", bytes.NewReader([]byte("hi")))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("status %d, want 415", resp.StatusCode)
	}
}

func TestReceiverRejectsBadBody(t *testing.T) {
	r, _ := startReceiver(t)

	resp, err := http.Post(r.Endpoint()+"/v1/traces", "application/x-protobuf", bytes.NewReader([]byte{0xff, 0xff}))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", resp.StatusCode)
	}
}
