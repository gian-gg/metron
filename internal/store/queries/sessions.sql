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
