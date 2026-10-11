package runner

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
)

func run(t *testing.T, cfg Config) (int, string) {
	t.Helper()
	var out bytes.Buffer
	if cfg.Stdout == nil {
		cfg.Stdout = &out
	}
	code, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("run %v: %v", cfg.Args, err)
	}
	return code, out.String()
}

func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   int
	}{
		{"success", "exit 0", 0},
		{"failure", "exit 3", 3},
		{"killed by signal", "kill -TERM $$", 128 + int(syscall.SIGTERM)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := run(t, Config{Args: []string{"sh", "-c", tt.script}}); got != tt.want {
				t.Errorf("got exit %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunPassesOutputAndEnv(t *testing.T) {
	_, out := run(t, Config{
		Args: []string{"sh", "-c", "echo $GREETING"},
		Env:  []string{"GREETING=hello"},
	})
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("got %q, want hello", out)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	if _, err := Run(context.Background(), Config{Args: []string{"metron-does-not-exist"}}); err == nil {
		t.Fatal("want error for unknown command")
	}
}

func TestRunNoArgsReturnsError(t *testing.T) {
	if _, err := Run(context.Background(), Config{}); err == nil {
		t.Fatal("want error for empty command")
	}
}

// runUntilReady starts script, sends sig once it prints "ready", and returns the exit code.
func runUntilReady(t *testing.T, script string, sig os.Signal) int {
	t.Helper()
	pr, pw := io.Pipe()
	signals := make(chan os.Signal, 1)

	go func() {
		line, _ := bufio.NewReader(pr).ReadString('\n')
		if strings.TrimSpace(line) == "ready" {
			signals <- sig
		}
		_, _ = io.Copy(io.Discard, pr)
	}()

	code, err := Run(context.Background(), Config{
		Args:    []string{"sh", "-c", script},
		Stdout:  pw,
		Signals: signals,
	})
	_ = pw.Close()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return code
}

func TestRunForwardsTerminate(t *testing.T) {
	script := `trap 'kill $!; exit 42' TERM; echo ready; sleep 10 >/dev/null 2>&1 & wait`

	if got := runUntilReady(t, script, syscall.SIGTERM); got != 42 {
		t.Errorf("got exit %d, want 42 from the TERM trap", got)
	}
}

func TestRunDoesNotForwardInterrupt(t *testing.T) {
	script := `trap 'exit 42' INT; echo ready; sleep 0.2 >/dev/null 2>&1 & wait; exit 0`

	if got := runUntilReady(t, script, os.Interrupt); got != 0 {
		t.Errorf("got exit %d, want 0 (interrupt must not be forwarded)", got)
	}
}

func TestCommandLine(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"python", "agent.py"}, "python agent.py"},
		{[]string{"python", "agent.py", "refund order 123"}, `python agent.py "refund order 123"`},
		{[]string{"echo", `say "hi"`}, `echo "say \"hi\""`},
		{[]string{"echo", ""}, `echo ""`},
	}
	for _, tt := range tests {
		if got := CommandLine(tt.args); got != tt.want {
			t.Errorf("CommandLine(%q) = %s, want %s", tt.args, got, tt.want)
		}
	}
}
