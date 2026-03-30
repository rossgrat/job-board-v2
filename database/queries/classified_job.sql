-- name: CreateClassifiedJob :one
INSERT INTO classified_job (id, raw_job_id, is_current)
VALUES ($1, $2, true)
RETURNING *;

-- name: GetClassifiedJobByID :one
SELECT * FROM classified_job WHERE id = $1;

-- name: UpdateClassifiedJobStatus :exec
UPDATE classified_job SET status = $2 WHERE id = $1;

-- name: ListClassifiedJobIDsByStatus :many
SELECT id FROM classified_job WHERE status = $1;

-- name: UpdateClassifiedJobNormalization :exec
UPDATE classified_job
SET title = $2, salary_min = $3, salary_max = $4, level = $5, normalized_at = now()
WHERE id = $1;

-- name: CreateClassifiedJobLocation :exec
INSERT INTO classified_job_location (id, classified_job_id, country, city, setting)
VALUES ($1, $2, $3, $4, $5);

-- name: CreateClassifiedJobTechnology :exec
INSERT INTO classified_job_technology (id, classified_job_id, name)
VALUES ($1, $2, $3);
