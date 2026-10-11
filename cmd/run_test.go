package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
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

	if !strings.Contains(out, "recorded") {
		t.Errorf("missing summary line:\n%s", out)
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
