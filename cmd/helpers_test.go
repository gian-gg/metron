package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

func executeErr(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func execute(t *testing.T, args ...string) string {
	t.Helper()
	out, err := executeErr(t, args...)
	if err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, out)
	}
	return out
}

func seed(t *testing.T, path string, sessions ...db.CreateSessionParams) {
	t.Helper()
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	for _, ses := range sessions {
		if err := s.CreateSession(context.Background(), ses); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}
}

func seedSpans(t *testing.T, path string, spans ...db.InsertSpanParams) {
	t.Helper()
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.InsertSpans(context.Background(), spans); err != nil {
		t.Fatalf("insert spans: %v", err)
	}
}

func span(sessionID, spanID, name, kind string, start int64) db.InsertSpanParams {
	return db.InsertSpanParams{
		TraceID: "5b8efff798038103d269b633813fc60c", SpanID: spanID, SessionID: sessionID,
		Name: name, Kind: kind, Status: "ok", StartTime: start, EndTime: start + 1_000_000, Attributes: "{}",
	}
}

// modelSpan is a span of kind model in its own trace, with token counts.
func modelSpan(sessionID, traceID, spanID string, start, end, in, out int64) db.InsertSpanParams {
	return db.InsertSpanParams{
		TraceID: traceID, SpanID: spanID, SessionID: sessionID, Name: "chat", Kind: "model", Status: "ok",
		StartTime: start, EndTime: end, InputTokens: &in, OutputTokens: &out, Attributes: "{}",
	}
}
