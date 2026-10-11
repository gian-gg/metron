package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gian-gg/metron/internal/store"
)

func onlyRun(t *testing.T, path string) (tag, caseName string, exitCode int64, finished bool) {
	t.Helper()
	runs := allRuns(t, path)
	if len(runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(runs))
	}
	r := runs[0]
	if r.Tag != nil {
		tag = *r.Tag
	}
	if r.CaseName != nil {
		caseName = *r.CaseName
	}
	if r.ExitCode != nil {
		exitCode = *r.ExitCode
	}
	return tag, caseName, exitCode, r.EndTime != nil
}

func TestRunRecordsFailureAndPassesExitCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")

	_, err := executeErr(t, "run", "--db", path, "--tag", "v2", "--case", "c1", "--", "sh", "-c", "exit 3")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Fatalf("got err %v, want ExitError with code 3", err)
	}
	tag, caseName, code, finished := onlyRun(t, path)
	if tag != "v2" || caseName != "c1" || code != 3 || !finished {
		t.Errorf("got tag=%q case=%q exit=%d finished=%v", tag, caseName, code, finished)
	}
}

func TestRunRecordsSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")

	out := execute(t, "run", "--db", path, "--", "true")

	if !strings.Contains(out, "recorded") || !strings.Contains(out, "0 spans") {
		t.Errorf("missing summary line with span count:\n%s", out)
	}
	if _, _, code, finished := onlyRun(t, path); code != 0 || !finished {
		t.Errorf("got exit=%d finished=%v, want 0 and finished", code, finished)
	}
}

func TestRunWithoutSeparatorPassesFlagsToCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")

	out := execute(t, "run", "--db", path, "--tag", "v1", "sh", "-c", "echo $0", "--tag")

	if !strings.Contains(out, "--tag") {
		t.Errorf("flag after command not passed through:\n%s", out)
	}
	if tag, _, _, _ := onlyRun(t, path); tag != "v1" {
		t.Errorf("got tag %q, want v1", tag)
	}
}

func TestRunUnknownCommandRecordsExit127(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")

	_, err := executeErr(t, "run", "--db", path, "--", "metron-does-not-exist")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 127 || exitErr.Err == nil {
		t.Fatalf("got err %v, want ExitError with code 127 and a cause", err)
	}
	if _, _, code, finished := onlyRun(t, path); code != 127 || !finished {
		t.Errorf("got exit=%d finished=%v, want 127 and finished", code, finished)
	}
}

func TestRunWithoutCommandFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")

	if _, err := executeErr(t, "run", "--db", path); err == nil {
		t.Fatal("want error when no command is given")
	}
}

const helperAgentEnv = "METRON_HELPER_AGENT"

// TestHelperAgent acts as an agent that exports two spans when run by TestRunRecordsSpans.
func TestHelperAgent(t *testing.T) {
	if os.Getenv(helperAgentEnv) != "1" {
		t.Skip("only runs as a helper process")
	}
	ctx := context.Background()
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		t.Fatalf("exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))

	ctx, parent := tp.Tracer("helper").Start(ctx, "agent")
	_, child := tp.Tracer("helper").Start(ctx, "chat")
	child.End()
	parent.End()

	if err := tp.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestRunRecordsSpans(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	t.Setenv(helperAgentEnv, "1")

	out := execute(t, "run", "--db", path, "--", os.Args[0], "-test.run=^TestHelperAgent$")

	if !strings.Contains(out, "2 spans") {
		t.Errorf("summary missing span count:\n%s", out)
	}
	runs := allRuns(t, path)
	if len(runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(runs))
	}

	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	var n int
	err = s.DB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM spans WHERE run_id = ?", runs[0].ID).Scan(&n)
	if err != nil {
		t.Fatalf("count spans: %v", err)
	}
	if n != 2 {
		t.Errorf("got %d spans for run, want 2", n)
	}
}
