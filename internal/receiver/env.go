package receiver

import "strings"

const (
	envEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envTracesEndpoint = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
)

// Env returns base with the OTLP endpoint pointed at endpoint, dropping any endpoint settings it overrides.
func Env(base []string, endpoint string) []string {
	out := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if strings.HasPrefix(kv, envEndpoint+"=") || strings.HasPrefix(kv, envTracesEndpoint+"=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, envEndpoint+"="+endpoint)
}
