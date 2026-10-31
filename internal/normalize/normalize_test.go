package normalize_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/gian-gg/metron/internal/normalize"
)

func ptr[T any](v T) *T { return &v }

func TestSpan(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]any
		want  normalize.Fields
	}{
		{
			name: "chat span with usage",
			attrs: map[string]any{
				"gen_ai.operation.name":      "chat",
				"gen_ai.request.model":       "gpt-5",
				"gen_ai.usage.input_tokens":  int64(120),
				"gen_ai.usage.output_tokens": int64(30),
			},
			want: normalize.Fields{Kind: "model", Model: ptr("gpt-5"), InputTokens: ptr(int64(120)), OutputTokens: ptr(int64(30))},
		},
		{
			name:  "model inferred from the model attribute alone",
			attrs: map[string]any{"gen_ai.request.model": "claude-x"},
			want:  normalize.Fields{Kind: "model", Model: ptr("claude-x")},
		},
		{
			name:  "response model used when the request model is absent",
			attrs: map[string]any{"gen_ai.response.model": "gpt-5-2026-01-01"},
			want:  normalize.Fields{Kind: "model", Model: ptr("gpt-5-2026-01-01")},
		},
		{
			name:  "request model wins over response model",
			attrs: map[string]any{"gen_ai.request.model": "a", "gen_ai.response.model": "b"},
			want:  normalize.Fields{Kind: "model", Model: ptr("a")},
		},
		{
			name:  "tool span by operation name",
			attrs: map[string]any{"gen_ai.operation.name": "execute_tool"},
			want:  normalize.Fields{Kind: "tool"},
		},
		{
			name:  "tool span by tool name",
			attrs: map[string]any{"gen_ai.tool.name": "calendar_read"},
			want:  normalize.Fields{Kind: "tool"},
		},
		{
			name: "tool wins when a tool span also carries a model",
			attrs: map[string]any{
				"gen_ai.tool.name":     "search",
				"gen_ai.request.model": "gpt-5",
			},
			want: normalize.Fields{Kind: "tool", Model: ptr("gpt-5")},
		},
		{
			name:  "model operation without a model attribute",
			attrs: map[string]any{"gen_ai.operation.name": "embeddings"},
			want:  normalize.Fields{Kind: "model"},
		},
		{
			name:  "non-model operation without a model attribute",
			attrs: map[string]any{"gen_ai.operation.name": "retrieval"},
			want:  normalize.Fields{Kind: "other"},
		},
		{
			name:  "agent span is not a model call",
			attrs: map[string]any{"gen_ai.operation.name": "invoke_agent", "gen_ai.request.model": "gpt-5"},
			want:  normalize.Fields{Kind: "other", Model: ptr("gpt-5")},
		},
		{
			name: "legacy token names",
			attrs: map[string]any{
				"gen_ai.request.model":           "m",
				"gen_ai.usage.prompt_tokens":     int64(7),
				"gen_ai.usage.completion_tokens": int64(3),
			},
			want: normalize.Fields{Kind: "model", Model: ptr("m"), InputTokens: ptr(int64(7)), OutputTokens: ptr(int64(3))},
		},
		{
			name: "current token names win over legacy ones",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  int64(10),
				"gen_ai.usage.prompt_tokens": int64(99),
			},
			want: normalize.Fields{Kind: "other", InputTokens: ptr(int64(10))},
		},
		{
			name: "tokens as float, string and zero",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  float64(42),
				"gen_ai.usage.output_tokens": "0",
			},
			want: normalize.Fields{Kind: "other", InputTokens: ptr(int64(42)), OutputTokens: ptr(int64(0))},
		},
		{
			name: "unusable token values give nil",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  int64(-5),
				"gen_ai.usage.output_tokens": "lots",
			},
			want: normalize.Fields{Kind: "other"},
		},
		{
			name: "fractional and boolean tokens give nil",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  1.5,
				"gen_ai.usage.output_tokens": true,
			},
			want: normalize.Fields{Kind: "other"},
		},
		{
			name:  "infinite and oversized floats give nil",
			attrs: map[string]any{"gen_ai.usage.input_tokens": math.Inf(1), "gen_ai.usage.output_tokens": 1e30},
			want:  normalize.Fields{Kind: "other"},
		},
		{
			name:  "empty model string gives nil",
			attrs: map[string]any{"gen_ai.request.model": ""},
			want:  normalize.Fields{Kind: "other"},
		},
		{
			name:  "no attributes",
			attrs: nil,
			want:  normalize.Fields{Kind: "other"},
		},
		{
			name:  "unrelated attributes are ignored",
			attrs: map[string]any{"step.kind": "tool", "cost.micro_cents": int64(1100)},
			want:  normalize.Fields{Kind: "other"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalize.Span(tt.attrs)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
