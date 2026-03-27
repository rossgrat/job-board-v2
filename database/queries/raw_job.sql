-- name: CreateRawJob :one
INSERT INTO raw_job (id, company_id, source_job_id, url, raw_data)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (company_id, source_job_id) DO NOTHING
RETURNING *;

-- name: GetRawJobByID :one
SELECT * FROM raw_job WHERE id = $1;

-- name: SetUserStatus :exec
UPDATE raw_job SET user_status = $2 WHERE id = $1;
