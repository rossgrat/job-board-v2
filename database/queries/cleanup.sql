-- name: DeleteStaleClassifiedJobs :execrows
DELETE FROM classified_job
WHERE is_current = false AND created_at < $1;

-- name: DeleteCompletedOutboxTasks :execrows
DELETE FROM outbox_task
WHERE status IN ('done', 'failed') AND updated_at < $1;
