-- name: CreateOutboxTask :one
INSERT INTO outbox_task (id, classified_job_id, task_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ClaimOutboxTask :one
SELECT * FROM outbox_task
WHERE task_name = $1
  AND status = 'waiting'
  AND (not_before IS NULL OR not_before <= now())
ORDER BY created_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: SetOutboxTaskStatus :exec
UPDATE outbox_task SET status = $2, updated_at = now() WHERE id = $1;

-- name: UpdateOutboxTask :exec
UPDATE outbox_task
SET status = $2, retry_count = $3, not_before = $4, updated_at = now()
WHERE id = $1;

-- name: ResetProcessingTasks :exec
UPDATE outbox_task SET status = 'waiting', updated_at = now()
WHERE status = 'processing';
