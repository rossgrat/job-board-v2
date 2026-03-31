package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	queries := db.New(s.pool)
	ctx := r.Context()

	cj, err := queries.GetClassifiedJobByID(ctx, id)
	if err != nil {
		slog.Error("failed to get classified job", slog.String("err", err.Error()))
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	rj, err := queries.GetRawJobByID(ctx, cj.RawJobID)
	if err != nil {
		slog.Error("failed to get raw job", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	company, err := queries.GetCompanyByID(ctx, rj.CompanyID)
	if err != nil {
		slog.Error("failed to get company", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	locations, err := queries.GetClassifiedJobLocations(ctx, cj.ID)
	if err != nil {
		slog.Error("failed to get locations", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	technologies, err := queries.GetClassifiedJobTechnologies(ctx, cj.ID)
	if err != nil {
		slog.Error("failed to get technologies", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	job := toJobDetail(cj, rj, company, locations, technologies)
	templates.JobDetailPage(job).Render(ctx, w)
}

func toJobDetail(
	cj db.ClassifiedJob,
	rj db.RawJob,
	company db.Company,
	locations []db.ClassifiedJobLocation,
	technologies []db.ClassifiedJobTechnology,
) templates.JobDetail {
	j := templates.JobDetail{
		ID:                          uuidToString(cj.ID),
		Status:                      cj.Status,
		IsCurrent:                   cj.IsCurrent,
		Title:                       textOrEmpty(cj.Title),
		Level:                       textOrEmpty(cj.Level),
		Category:                    textOrEmpty(cj.Category),
		Relevance:                   textOrEmpty(cj.Relevance),
		Reasoning:                   textOrEmpty(cj.Reasoning),
		ClassificationPromptVersion: textOrEmpty(cj.ClassificationPromptVersion),

		RawJobID:    uuidToString(rj.ID),
		URL:         rj.Url,
		SourceJobID: rj.SourceJobID,
		UserStatus:  textOrEmpty(rj.UserStatus),
		CleanData:   rj.CleanData,
		RawData:     string(rj.RawData),

		CompanyName:    company.Name,
		CompanyFavicon: company.FaviconUrl,
	}

	if cj.SalaryMin.Valid && cj.SalaryMax.Valid {
		j.HasSalary = true
		j.SalaryMin = cj.SalaryMin.Int32
		j.SalaryMax = cj.SalaryMax.Int32
	}

	j.CreatedAt = formatTimestamp(cj.CreatedAt)
	j.NormalizedAt = formatTimestamp(cj.NormalizedAt)
	j.ClassifiedAt = formatTimestamp(cj.ClassifiedAt)
	j.DiscoveredAt = formatTimestamp(rj.DiscoveredAt)

	for _, loc := range locations {
		j.Locations = append(j.Locations, templates.Location{
			Setting: loc.Setting,
			Country: loc.Country,
			City:    textOrEmpty(loc.City),
		})
	}

	for _, tech := range technologies {
		j.Technologies = append(j.Technologies, tech.Name)
	}

	return j
}

func formatTimestamp(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05 MST")
}
