package receiver

import (
	"testing"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func str(s string) *commonpb.AnyValue {
	return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: s}}
}

func TestAttributesJSON(t *testing.T) {
	span := []*commonpb.KeyValue{
		{Key: "s", Value: str("x")},
		{Key: "b", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: true}}},
		{Key: "i", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 7}}},
		{Key: "d", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 1.5}}},
		{Key: "bytes", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BytesValue{BytesValue: []byte("hi")}}},
		{Key: "arr", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_ArrayValue{
			ArrayValue: &commonpb.ArrayValue{Values: []*commonpb.AnyValue{str("a"), str("b")}},
		}}},
		{Key: "kv", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_KvlistValue{
			KvlistValue: &commonpb.KeyValueList{Values: []*commonpb.KeyValue{{Key: "k", Value: str("v")}}},
		}}},
		{Key: "empty", Value: nil},
	}
	resource := []*commonpb.KeyValue{{Key: "service.name", Value: str("agent")}}

	got, err := attributesJSON(span, resource)
	if err != nil {
		t.Fatalf("attributesJSON: %v", err)
	}

	want := `{"resource":{"service.name":"agent"},"span":{"arr":["a","b"],"b":true,"bytes":"aGk=","d":1.5,"empty":null,"i":7,"kv":{"k":"v"},"s":"x"}}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestStatusName(t *testing.T) {
	tests := []struct {
		status *tracepb.Status
		want   string
	}{
		{nil, "unset"},
		{&tracepb.Status{Code: tracepb.Status_STATUS_CODE_UNSET}, "unset"},
		{&tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK}, "ok"},
		{&tracepb.Status{Code: tracepb.Status_STATUS_CODE_ERROR}, "error"},
	}
	for _, tt := range tests {
		if got := statusName(tt.status); got != tt.want {
			t.Errorf("statusName(%v) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestOptionalHex(t *testing.T) {
	if got := optionalHex(nil); got != nil {
		t.Errorf("empty id = %v, want nil", *got)
	}
	if got := optionalHex([]byte{0xab, 0x01}); got == nil || *got != "ab01" {
		t.Errorf("got %v, want ab01", got)
	}
}
