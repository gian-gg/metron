package cmd

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/store"
	"github.com/gian-gg/metron/internal/store/db"
)

const serveWait = 5 * time.Second

const jsonTrace = `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"agent"}}]},
"scopeSpans":[{"spans":[{"traceId":"5b8efff798038103d269b633813fc60c","spanId":"eee19b7ec3c1b174",
"name":"chat","startTimeUnixNano":"1000","endTimeUnixNano":"2000","status":{"code":1}}]}]}]}`

var listenAddr = regexp.MustCompile(`127\.0\.0\.1:\d+`)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runServe starts metron serve in the background and returns once it is listening.
func runServe(t *testing.T, args ...string) (stderr *syncBuffer, addr string, stop func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	root := NewRootCmd()
	stderr = &syncBuffer{}
	root.SetErr(stderr)
	root.SetArgs(append([]string{"serve", "--port", "0", "--grpc-port", "0"}, args...))

	errc := make(chan error, 1)
	go func() { errc <- root.ExecuteContext(ctx) }()

	deadline := time.Now().Add(serveWait)
	for time.Now().Before(deadline) {
		if m := listenAddr.FindString(stderr.String()); m != "" {
			return stderr, m, func() error {
				cancel()
				select {
				case err := <-errc:
					return err
				case <-time.After(serveWait):
					t.Fatal("serve did not stop")
					return nil
				}
			}
		}
		select {
		case err := <-errc:
			t.Fatalf("serve exited early: %v\n%s", err, stderr.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatalf("serve never started listening:\n%s", stderr.String())
	return nil, "", nil
}

func listSessions(t *testing.T, path string) (*store.Store, []db.Session) {
	t.Helper()
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	sessions, err := s.ListSessions(context.Background(), db.ListSessionsParams{Limit: 100})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	return s, sessions
}

func TestServeRecordsSpansUnderATaggedSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	stderr, addr, stop := runServe(t, "--db", path, "--tag", "v1")

	resp, err := http.Post("http://"+addr+"/v1/traces", "application/json", strings.NewReader(jsonTrace))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	if err := stop(); err != nil {
		t.Fatalf("serve: %v", err)
	}

	s, sessions := listSessions(t, path)
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	session := sessions[0]
	if session.Tag == nil || *session.Tag != "v1" {
		t.Errorf("session tag = %v, want v1", session.Tag)
	}
	if session.EndTime == nil {
		t.Error("session has no end time after shutdown")
	}
	var spans int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM spans WHERE session_id = ?`, session.ID).Scan(&spans); err != nil {
		t.Fatalf("count spans: %v", err)
	}
	if spans != 1 {
		t.Errorf("session has %d spans, want 1", spans)
	}
	if !strings.Contains(stderr.String(), "tag v1") || !strings.Contains(stderr.String(), "stopped · 1 span") {
		t.Errorf("stderr missing start or stop line:\n%s", stderr.String())
	}
}

func TestServeFailsOnTakenPortAndLeavesNoSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metron.db")
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = taken.Close() }()
	_, port, _ := net.SplitHostPort(taken.Addr().String())

	_, err = executeErr(t, "serve", "--db", path, "--port", port, "--grpc-port", "0")

	if err == nil || !strings.Contains(err.Error(), "--port") {
		t.Fatalf("err = %v, want one pointing at --port", err)
	}
	if _, sessions := listSessions(t, path); len(sessions) != 0 {
		t.Errorf("got %d sessions, want none left behind", len(sessions))
	}
}

func TestListenAddrs(t *testing.T) {
	got := listenAddrs(serveOptions{listen: "0.0.0.0", httpPort: 4318, grpcPort: 4317})

	want := []string{"0.0.0.0:4318", "0.0.0.0:4317"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}
