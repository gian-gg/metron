package receiver

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/gian-gg/metron/internal/store/db"
)

// toRows converts an OTLP export into span rows for runID.
func toRows(runID string, req *coltracepb.ExportTraceServiceRequest) ([]db.InsertSpanParams, error) {
	var rows []db.InsertSpanParams
	for _, rs := range req.GetResourceSpans() {
		resource := rs.GetResource().GetAttributes()
		for _, ss := range rs.GetScopeSpans() {
			for _, span := range ss.GetSpans() {
				attrs, err := attributesJSON(span.GetAttributes(), resource)
				if err != nil {
					return nil, err
				}
				rows = append(rows, db.InsertSpanParams{
					TraceID:      hex.EncodeToString(span.GetTraceId()),
					SpanID:       hex.EncodeToString(span.GetSpanId()),
					ParentSpanID: optionalHex(span.GetParentSpanId()),
					RunID:        &runID,
					Name:         span.GetName(),
					Status:       statusName(span.GetStatus()),
					StartTime:    toInt64(span.GetStartTimeUnixNano()),
					EndTime:      toInt64(span.GetEndTimeUnixNano()),
					Attributes:   attrs,
				})
			}
		}
	}
	return rows, nil
}

func optionalHex(id []byte) *string {
	if len(id) == 0 {
		return nil
	}
	s := hex.EncodeToString(id)
	return &s
}

func statusName(s *tracepb.Status) string {
	switch s.GetCode() {
	case tracepb.Status_STATUS_CODE_OK:
		return "ok"
	case tracepb.Status_STATUS_CODE_ERROR:
		return "error"
	default:
		return "unset"
	}
}

func toInt64(ns uint64) int64 {
	if ns > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(ns)
}

func attributesJSON(span, resource []*commonpb.KeyValue) (string, error) {
	out, err := json.Marshal(map[string]any{
		"span":     keyValues(span),
		"resource": keyValues(resource),
	})
	if err != nil {
		return "", fmt.Errorf("encode attributes: %w", err)
	}
	return string(out), nil
}

func keyValues(kvs []*commonpb.KeyValue) map[string]any {
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		out[kv.GetKey()] = anyValue(kv.GetValue())
	}
	return out
}

func anyValue(v *commonpb.AnyValue) any {
	switch val := v.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return val.StringValue
	case *commonpb.AnyValue_BoolValue:
		return val.BoolValue
	case *commonpb.AnyValue_IntValue:
		return val.IntValue
	case *commonpb.AnyValue_DoubleValue:
		if math.IsNaN(val.DoubleValue) || math.IsInf(val.DoubleValue, 0) {
			return strconv.FormatFloat(val.DoubleValue, 'g', -1, 64)
		}
		return val.DoubleValue
	case *commonpb.AnyValue_BytesValue:
		return val.BytesValue
	case *commonpb.AnyValue_ArrayValue:
		items := make([]any, len(val.ArrayValue.GetValues()))
		for i, item := range val.ArrayValue.GetValues() {
			items[i] = anyValue(item)
		}
		return items
	case *commonpb.AnyValue_KvlistValue:
		return keyValues(val.KvlistValue.GetValues())
	default:
		return nil
	}
}
