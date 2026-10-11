package receiver

import (
	"slices"
	"testing"
)

func TestEnv(t *testing.T) {
	base := []string{
		"PATH=/bin",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://elsewhere:4318",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://elsewhere:4318/v1/traces",
		"OTEL_SERVICE_NAME=agent",
	}
	original := slices.Clone(base)

	got := Env(base, "http://127.0.0.1:9999")

	want := []string{
		"PATH=/bin",
		"OTEL_SERVICE_NAME=agent",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:9999",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if !slices.Equal(base, original) {
		t.Errorf("input modified: %q", base)
	}
}
