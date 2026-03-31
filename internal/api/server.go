package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	pool       *pgxpool.Pool
	cfg        *config.Config
	r          *runner.Runner
}

func New(pool *pgxpool.Pool, cfg *config.Config) *Server {
	s := &Server{
		router: chi.NewRouter(),
		pool:   pool,
		cfg:    cfg,
	}

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.routes()

	s.httpServer = &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: s.router,
	}

	s.r = runner.New(
		runner.WithProcess(s.serverRunner()),
	)

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
