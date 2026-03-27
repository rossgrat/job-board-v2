-- name: CreateClassifiedJob :one
INSERT INTO classified_job (id, raw_job_id, is_current)
VALUES ($1, $2, true)
RETURNING *;
