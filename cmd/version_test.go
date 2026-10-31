package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	info := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	none := func() (*debug.BuildInfo, bool) { return nil, false }
	tests := []struct {
		name    string
		version string
		read    func() (*debug.BuildInfo, bool)
		want    string
	}{
		{"release build keeps its injected version", "0.1.0", info("v9.9.9"), "0.1.0"},
		{"go install uses the module version", "dev", info("v0.1.0"), "v0.1.0"},
		{"local build stays dev", "dev", info("(devel)"), "dev"},
		{"empty module version stays dev", "dev", info(""), "dev"},
		{"no build info stays dev", "dev", none, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.version, tt.read); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
