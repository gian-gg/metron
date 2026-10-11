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

func seed(t *testing.T, path string, runs ...db.CreateRunParams) {
	t.Helper()
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	for _, r := range runs {
		if err := s.CreateRun(context.Background(), r); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}
}

func allRuns(t *testing.T, path string) []db.Run {
	t.Helper()
	s, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = s.Close() }()
	runs, err := s.ListRuns(context.Background(), db.ListRunsParams{Limit: 100})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	return runs
}
