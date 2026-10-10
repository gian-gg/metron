-- name: CreateRun :exec
INSERT INTO runs (id, tag, case_name, command, start_time)
VALUES (?, ?, ?, ?, ?);

-- name: FinishRun :exec
UPDATE runs
SET end_time = ?, exit_code = ?
WHERE id = ?;

-- name: ListRuns :many
SELECT * FROM runs
ORDER BY start_time DESC
LIMIT ?;
