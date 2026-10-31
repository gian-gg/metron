package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestListening(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"with tag", "v2", "● listening on 127.0.0.1:4318, 127.0.0.1:4317 · tag v2"},
		{"without tag", "", "● listening on 127.0.0.1:4318, 127.0.0.1:4317\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := Listening(&buf, []string{"127.0.0.1:4318", "127.0.0.1:4317"}, tt.tag)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("got %q, want it to contain %q", buf.String(), tt.want)
			}
			if strings.Contains(buf.String(), "\x1b[") {
				t.Errorf("output contains color codes: %q", buf.String())
			}
		})
	}
}

func TestSpanCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0 spans"},
		{1, "1 span"},
		{6, "6 spans"},
	}
	for _, tt := range tests {
		if got := spanCount(tt.n); got != tt.want {
			t.Errorf("spanCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestStopped(t *testing.T) {
	var buf bytes.Buffer

	if err := Stopped(&buf, 6); err != nil {
		t.Fatalf("render: %v", err)
	}

	if got := buf.String(); got != "● stopped · 6 spans\n" {
		t.Errorf("got %q", got)
	}
}
