-- name: CreateClassifiedJob :one
INSERT INTO classified_job (id, raw_job_id, is_current)
VALUES ($1, $2, true)
RETURNING *;

-- name: GetClassifiedJobByID :one
SELECT * FROM classified_job WHERE id = $1;

-- name: UpdateClassifiedJobStatus :exec
UPDATE classified_job SET status = $2 WHERE id = $1;
