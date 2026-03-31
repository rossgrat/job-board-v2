package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
	queries := db.New(s.pool)

	rows, err := queries.ListCompanies(r.Context())
	if err != nil {
		slog.Error("failed to list companies", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	companies := make([]templates.CompanyItem, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, templates.CompanyItem{
			ID:        uuidToString(row.ID),
			Name:      row.Name,
			Favicon:   row.FaviconUrl,
			FetchType: row.FetchType,
			IsActive:  row.IsActive,
		})
	}

	templates.CompaniesPage(companies).Render(r.Context(), w)
}

func (s *Server) handleCompanyToggle(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	queries := db.New(s.pool)
	ctx := r.Context()

	company, err := queries.GetCompanyByID(ctx, id)
	if err != nil {
		slog.Error("failed to get company", slog.String("err", err.Error()))
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	newActive := !company.IsActive
	err = queries.SetCompanyActive(ctx, db.SetCompanyActiveParams{
		ID:       id,
		IsActive: newActive,
	})
	if err != nil {
		slog.Error("failed to toggle company", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	item := templates.CompanyItem{
		ID:        uuidToString(id),
		Name:      company.Name,
		Favicon:   company.FaviconUrl,
		FetchType: company.FetchType,
		IsActive:  newActive,
	}

	templates.CompanyItemFragment(item).Render(ctx, w)
}
