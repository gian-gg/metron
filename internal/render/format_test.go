package render

import (
	"testing"
	"time"
)

func TestShortID(t *testing.T) {
	tests := []struct{ id, want string }{
		{"a1b2c3d4e5f6", "a1b2c3d4"},
		{"abc", "abc"},
	}
	for _, tt := range tests {
		if got := shortID(tt.id); got != tt.want {
			t.Errorf("shortID(%q) = %q, want %q", tt.id, got, tt.want)
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
		{"nil", nil, placeholder},
		{"empty", &empty, placeholder},
		{"value", &v2, "v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := orPlaceholder(tt.in); got != tt.want {
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
		if got := ago(tt.d); got != tt.want {
			t.Errorf("ago(%v) = %q, want %q", tt.d, got, tt.want)
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
		if got := duration(tt.d); got != tt.want {
			t.Errorf("duration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
