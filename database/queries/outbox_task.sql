-- name: CreateOutboxTask :one
INSERT INTO outbox_task (id, classified_job_id, task_name)
VALUES ($1, $2, $3)
RETURNING *;
