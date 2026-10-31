-- name: CreateSession :exec
INSERT INTO sessions (id, tag, start_time)
VALUES (?, ?, ?);

-- name: FinishSession :exec
UPDATE sessions
SET end_time = ?
WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions
WHERE tag = sqlc.narg(tag) OR sqlc.narg(tag) IS NULL
ORDER BY start_time DESC
LIMIT sqlc.arg(limit);

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: SessionsByPrefix :many
SELECT * FROM sessions
WHERE substr(id, 1, length(CAST(sqlc.arg(prefix) AS TEXT))) = CAST(sqlc.arg(prefix) AS TEXT)
LIMIT 2;

-- name: SessionsByTag :many
SELECT * FROM sessions
WHERE tag = ?
ORDER BY start_time;

-- name: ListGroups :many
SELECT
    CAST(COALESCE(s.tag, s.id) AS TEXT)                                AS name,
    CAST(MAX(s.tag IS NOT NULL) AS INTEGER)                            AS tagged,
    CAST(COUNT(DISTINCT s.id) AS INTEGER)                              AS sessions,
    CAST(COUNT(DISTINCT sp.trace_id) AS INTEGER)                       AS traces,
    CAST(MIN(s.start_time) AS INTEGER)                                 AS started,
    CAST(MAX(COALESCE(sp.end_time, s.end_time, s.start_time)) AS INTEGER) AS last_active,
    CAST(MAX(s.end_time IS NULL) AS INTEGER)                           AS running
FROM sessions s
LEFT JOIN spans sp ON sp.session_id = s.id
WHERE s.tag = sqlc.narg(tag) OR sqlc.narg(tag) IS NULL
GROUP BY COALESCE(s.tag, s.id)
ORDER BY started DESC
LIMIT sqlc.arg(limit);
