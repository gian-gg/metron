-- name: InsertSpan :exec
INSERT INTO spans (trace_id, span_id, parent_span_id, session_id, name, kind, status, start_time, end_time, model, input_tokens, output_tokens, attributes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (trace_id, span_id) DO NOTHING;

-- name: ListSessionSpans :many
SELECT trace_id, span_id, parent_span_id, name, kind, status, start_time, end_time, model, input_tokens, output_tokens
FROM spans
WHERE session_id = ?
ORDER BY start_time, span_id;

-- name: ListSessionTraceStats :many
SELECT
    trace_id,
    CAST(MIN(start_time) AS INTEGER)                                                   AS started,
    CAST(MAX(end_time) AS INTEGER)                                                     AS ended,
    CAST(SUM(status = 'error') AS INTEGER)                                             AS errors,
    CAST(SUM(kind = 'model') AS INTEGER)                                               AS model_calls,
    CAST(SUM(kind = 'tool') AS INTEGER)                                                AS tool_calls,
    CAST(COALESCE(SUM(CASE WHEN kind = 'model' THEN input_tokens END), 0) AS INTEGER)  AS input_tokens,
    CAST(COALESCE(SUM(CASE WHEN kind = 'model' THEN output_tokens END), 0) AS INTEGER) AS output_tokens
FROM spans
WHERE session_id = ?
GROUP BY trace_id;
