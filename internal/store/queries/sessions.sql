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
