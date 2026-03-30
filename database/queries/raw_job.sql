-- name: CreateRawJob :one
INSERT INTO raw_job (id, company_id, source_job_id, url, raw_data, clean_data)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (company_id, source_job_id) DO NOTHING
RETURNING *;

-- name: GetRawJobByID :one
SELECT * FROM raw_job WHERE id = $1;

-- name: SetUserStatus :exec
UPDATE raw_job SET user_status = $2 WHERE id = $1;

-- name: GetRawJobsWithEmptyCleanData :many
SELECT * FROM raw_job WHERE clean_data = '';

-- name: UpdateRawJobCleanData :exec
UPDATE raw_job SET clean_data = $2 WHERE id = $1;
