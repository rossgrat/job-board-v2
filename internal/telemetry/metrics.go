package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("job-board")

	fetcherJobsFetched metric.Int64Counter
	fetcherJobsSaved   metric.Int64Counter
	fetcherErrors      metric.Int64Counter

	outboxTasksCompleted metric.Int64Counter
	outboxTaskDuration   metric.Float64Histogram
)

func init() {
	fetcherJobsFetched, _ = meter.Int64Counter("fetcher.jobs_fetched")
	fetcherJobsSaved, _ = meter.Int64Counter("fetcher.jobs_saved")
	fetcherErrors, _ = meter.Int64Counter("fetcher.errors")

	outboxTasksCompleted, _ = meter.Int64Counter("outbox.tasks_completed")
	outboxTaskDuration, _ = meter.Float64Histogram("outbox.task_duration_ms")
}

func RecordJobsFetched(ctx context.Context, company string, count int64) {
	fetcherJobsFetched.Add(ctx, count, metric.WithAttributes(attribute.String("company", company)))
}

func RecordJobSaved(ctx context.Context, company string) {
	fetcherJobsSaved.Add(ctx, 1, metric.WithAttributes(attribute.String("company", company)))
}

func RecordFetchError(ctx context.Context, company string) {
	fetcherErrors.Add(ctx, 1, metric.WithAttributes(attribute.String("company", company)))
}

func RecordTaskCompleted(ctx context.Context, taskName, status string) {
	outboxTasksCompleted.Add(ctx, 1, metric.WithAttributes(
		attribute.String("task_name", taskName),
		attribute.String("status", status),
	))
}

func RecordTaskDuration(ctx context.Context, taskName string, durationMs float64) {
	outboxTaskDuration.Record(ctx, durationMs, metric.WithAttributes(attribute.String("task_name", taskName)))
}
