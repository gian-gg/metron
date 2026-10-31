package ui_test

import (
	"bytes"
	"testing"

	"github.com/gian-gg/metron/internal/render/internal/ui"
	"github.com/gian-gg/metron/internal/store/db"
)

func TestStatus(t *testing.T) {
	end := int64(1)
	tests := []struct {
		name    string
		session db.Session
		want    string
	}{
		{"running", db.Session{}, "running"},
		{"done", db.Session{EndTime: &end}, "done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.SessionStatus(tt.session); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusCellIcons(t *testing.T) {
	end := int64(1)
	st := ui.NewStyles(&bytes.Buffer{})
	tests := []struct {
		name    string
		session db.Session
		want    string
	}{
		{"running", db.Session{}, "● running"},
		{"done", db.Session{EndTime: &end}, "✓ done"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.SessionStatusCell(tt.session, st); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
