-- name: CreateRawJob :one
INSERT INTO raw_job (id, company_id, source_job_id, url, raw_data, clean_data)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (company_id, source_job_id) DO UPDATE SET deleted_at = NULL WHERE raw_job.deleted_at IS NOT NULL
RETURNING *;

-- name: GetRawJobByID :one
SELECT * FROM raw_job WHERE id = $1;

-- name: SetUserStatus :exec
UPDATE raw_job SET user_status = $2, rejection_reason = $3 WHERE id = $1;

-- name: GetRawJobsWithEmptyCleanData :many
SELECT * FROM raw_job WHERE clean_data = '';

-- name: UpdateRawJobCleanData :exec
UPDATE raw_job SET clean_data = $2 WHERE id = $1;

-- name: SoftDeleteMissingJobs :execrows
UPDATE raw_job
SET deleted_at = now()
WHERE company_id = @company_id
  AND deleted_at IS NULL
  AND source_job_id != ALL(@seen_source_job_ids::text[]);
