package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otellog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Telemetry struct {
	meterProvider *metric.MeterProvider
	logProvider   *otellog.LoggerProvider
	Logger        *slog.Logger
}

func Init(ctx context.Context, endpoint string) (*Telemetry, error) {
	// Metrics
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(15*time.Second))),
	)
	otel.SetMeterProvider(meterProvider)

	// Logs
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	logProvider := otellog.NewLoggerProvider(
		otellog.WithProcessor(otellog.NewBatchProcessor(logExporter)),
	)

	otelHandler := otelslog.NewHandler("job-board", otelslog.WithLoggerProvider(logProvider))
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(newMultiHandler(jsonHandler, otelHandler))

	return &Telemetry{
		meterProvider: meterProvider,
		logProvider:   logProvider,
		Logger:        logger,
	}, nil
}

func (t *Telemetry) NewTelemetryCloser() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			slog.Info("shutting down telemetry")
			t.meterProvider.Shutdown(ctx)
			t.logProvider.Shutdown(ctx)
			return nil
		}
	}
}

// multiHandler fans out log records to multiple slog handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			handler.Handle(ctx, r)
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return newMultiHandler(handlers...)
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(handlers...)
}
