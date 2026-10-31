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
