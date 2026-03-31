package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/telemetry"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	pool       *pgxpool.Pool
	cfg        *config.Config
	r          *runner.Runner
}

func New(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) *Server {

	// Init telemetry
	tel, err := telemetry.Init(ctx, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		slog.Warn("telemetry init failed, continuing without", slog.String("err", err.Error()))
	} else {
		slog.SetDefault(tel.Logger)
	}

	s := &Server{
		router: chi.NewRouter(),
		pool:   pool,
		cfg:    cfg,
	}

	s.router.Use(slogRequestLogger)
	s.router.Use(middleware.Recoverer)
	s.routes()

	s.httpServer = &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: s.router,
	}

	runnerOpts := []runner.RunnerOption{
		runner.WithProcess(s.serverRunner()),
	}

	if tel != nil {
		runnerOpts = append(runnerOpts, runner.WithCloser(tel.NewTelemetryCloser()))
	}

	s.r = runner.New(runnerOpts...)

	return s
}

func (s *Server) Run() {
	s.r.Run()
}

func (s *Server) serverRunner() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			slog.Info("starting API server", slog.String("addr", s.httpServer.Addr))

			errCh := make(chan error, 1)
			go func() {
				if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errCh <- err
				}
				close(errCh)
			}()

			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				slog.Info("shutting down API server")
				return s.httpServer.Shutdown(context.Background())
			}
		}
	}
}
