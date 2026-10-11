package receiver

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	maxBodyBytes    = 32 << 20
	contentProtobuf = "application/x-protobuf"
	contentJSON     = "application/json"
)

func (r *Receiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || (mediaType != contentProtobuf && mediaType != contentJSON) {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := readBody(w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	export, err := decodeExport(mediaType, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.accept(req.Context(), export); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeResponse(w, mediaType)
}

func readBody(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	var body io.Reader = http.MaxBytesReader(w, req.Body, maxBodyBytes)
	if req.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("read gzip body: %w", err)
		}
		defer func() { _ = gz.Close() }()
		body = io.LimitReader(gz, maxBodyBytes)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return data, nil
}

func decodeExport(mediaType string, body []byte) (*coltracepb.ExportTraceServiceRequest, error) {
	export := &coltracepb.ExportTraceServiceRequest{}
	if mediaType == contentProtobuf {
		if err := proto.Unmarshal(body, export); err != nil {
			return nil, fmt.Errorf("decode protobuf: %w", err)
		}
		return export, nil
	}

	body, err := hexIDsToBase64(body)
	if err != nil {
		return nil, err
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, export); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return export, nil
}

func writeResponse(w http.ResponseWriter, mediaType string) {
	resp := &coltracepb.ExportTraceServiceResponse{}
	var out []byte
	if mediaType == contentProtobuf {
		out, _ = proto.Marshal(resp)
	} else {
		out, _ = protojson.Marshal(resp)
	}
	w.Header().Set("Content-Type", mediaType)
	_, _ = w.Write(out)
}

var idFields = map[string]bool{"traceId": true, "spanId": true, "parentSpanId": true}

// hexIDsToBase64 rewrites OTLP/JSON hex trace and span IDs into the base64 form protojson expects.
func hexIDsToBase64(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return json.Marshal(rewriteIDs(doc))
}

func rewriteIDs(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if s, ok := val.(string); ok && idFields[k] {
				out[k] = hexToBase64(s)
				continue
			}
			out[k] = rewriteIDs(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = rewriteIDs(val)
		}
		return out
	default:
		return v
	}
}

func hexToBase64(s string) string {
	b, err := hex.DecodeString(s)
	if err != nil {
		return s
	}
	return base64.StdEncoding.EncodeToString(b)
}
