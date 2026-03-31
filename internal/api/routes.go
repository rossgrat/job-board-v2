package api

import "net/http"

func (s *Server) routes() {
	s.router.Get("/", s.handleDashboard)
	s.router.Get("/browse", s.handleFilter)
	s.router.Get("/jobs/{id}", s.handleJobDetail)
	s.router.Get("/jobs/{id}/review", s.handleReviewModal)
	s.router.Post("/jobs/{id}/status", s.handleSetStatus)
	s.router.Post("/jobs/{id}/eval", s.handleSetEval)
	s.router.Get("/companies", s.handleCompanies)
	s.router.Post("/companies/{id}/toggle", s.handleCompanyToggle)

	fs := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fs))
}
