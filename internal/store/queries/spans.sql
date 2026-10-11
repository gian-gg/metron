-- name: InsertSpan :exec
INSERT INTO spans (trace_id, span_id, parent_span_id, run_id, name, status, start_time, end_time, attributes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (trace_id, span_id) DO NOTHING;
