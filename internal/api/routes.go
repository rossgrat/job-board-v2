package api

import "net/http"

func (s *Server) routes() {
	s.router.Get("/", s.handleDashboard)
	s.router.Get("/jobs/{id}", s.handleJobDetail)

	fs := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fs))
}
