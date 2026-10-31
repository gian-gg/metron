package ui_test

import (
	"testing"
	"time"

	"github.com/gian-gg/metron/internal/render/internal/ui"
)

func TestShortID(t *testing.T) {
	tests := []struct{ id, want string }{
		{"a1b2c3d4e5f6", "a1b2c3d4"},
		{"abc", "abc"},
	}
	for _, tt := range tests {
		if got := ui.ShortID(tt.id); got != tt.want {
			t.Errorf("ui.ShortID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestOrPlaceholder(t *testing.T) {
	empty, v2 := "", "v2"
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, ui.Placeholder},
		{"empty", &empty, ui.Placeholder},
		{"value", &v2, "v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.OrPlaceholder(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgo(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s ago"},
		{2 * time.Minute, "2m ago"},
		{3 * time.Hour, "3h ago"},
		{50 * time.Hour, "2d ago"},
	}
	for _, tt := range tests {
		if got := ui.Ago(tt.d); got != tt.want {
			t.Errorf("ui.Ago(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{850 * time.Millisecond, "850ms"},
		{5120 * time.Millisecond, "5.1s"},
		{125 * time.Second, "2m5s"},
	}
	for _, tt := range tests {
		if got := ui.Duration(tt.d); got != tt.want {
			t.Errorf("ui.Duration(%v) = %q, want %q", tt.d, got, tt.want)
		}
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
		if got := ui.SpanCount(tt.n); got != tt.want {
			t.Errorf("ui.SpanCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestSessionCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{{1, "1 session"}, {3, "3 sessions"}}
	for _, tt := range tests {
		if got := ui.SessionCount(tt.n); got != tt.want {
			t.Errorf("ui.SessionCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
