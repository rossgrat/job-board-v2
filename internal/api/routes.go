package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) routes() {
	// Public routes
	s.router.Get("/login", s.handleLoginPage)
	s.router.Post("/login", s.handleLogin)
	s.router.Post("/logout", s.handleLogout)

	fs := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fs))

	// Serve service worker from root so it can control all pages
	s.router.Get("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, "static/service-worker.js")
	})

	// Authenticated routes
	s.router.Group(func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.Auth.Password))

		r.Get("/", s.handleDashboard)
		r.Get("/browse", s.handleFilter)
		r.Get("/jobs/{id}", s.handleJobDetail)
		r.Get("/jobs/{id}/review", s.handleReviewModal)
		r.Post("/jobs/{id}/status", s.handleSetStatus)
		r.Post("/jobs/{id}/eval", s.handleSetEval)
		r.Get("/companies", s.handleCompanies)
		r.Post("/companies/{id}/toggle", s.handleCompanyToggle)
	})
}
