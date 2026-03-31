-- name: UpsertEvalEntry :exec
INSERT INTO eval_entry (id, raw_job_id, expected_category, expected_relevance)
VALUES ($1, $2, $3, $4)
ON CONFLICT (raw_job_id) DO UPDATE
SET expected_category = $3, expected_relevance = $4;

-- name: GetEvalEntryByRawJobID :one
SELECT * FROM eval_entry WHERE raw_job_id = $1;
