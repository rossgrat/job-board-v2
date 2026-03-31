package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleFilter(w http.ResponseWriter, r *http.Request) {
	relevance := r.URL.Query().Get("relevance")
	userStatus := r.URL.Query().Get("user_status")
	companyName := r.URL.Query().Get("company")

	queries := db.New(s.pool)
	ctx := r.Context()

	rows, err := queries.ListFilteredJobs(ctx, db.ListFilteredJobsParams{
		Relevance:   relevance,
		UserStatus:  userStatus,
		CompanyName: companyName,
	})
	if err != nil {
		slog.Error("failed to list filtered jobs", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	locFilters, err := s.loadLocationFilters(ctx)
	if err != nil {
		slog.Error("failed to load location filters", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jobs := make([]templates.DashboardJob, 0, len(rows))
	for _, row := range rows {
		j := toFilteredJob(row)
		j.Locations = filterLocations(j.Locations, locFilters)
		jobs = append(jobs, j)
	}

	companies, err := queries.ListCompanies(ctx)
	if err != nil {
		slog.Error("failed to list companies", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	companyNames := make([]string, 0, len(companies))
	for _, c := range companies {
		companyNames = append(companyNames, c.Name)
	}

	filters := templates.FilterState{
		Relevance:    relevance,
		UserStatus:   userStatus,
		CompanyName:  companyName,
		CompanyNames: companyNames,
	}

	templates.FilterPage(jobs, filters).Render(ctx, w)
}

func toFilteredJob(row db.ListFilteredJobsRow) templates.DashboardJob {
	j := templates.DashboardJob{
		ID:             uuidToString(row.ID),
		Title:          textOrEmpty(row.Title),
		URL:            row.Url,
		CompanyName:    row.CompanyName,
		CompanyFavicon: row.CompanyFaviconUrl,
		Level:          textOrEmpty(row.Level),
		Category:       textOrEmpty(row.Category),
		Relevance:      textOrEmpty(row.Relevance),
		UserStatus:     textOrEmpty(row.UserStatus),
	}

	if row.SalaryMin.Valid && row.SalaryMax.Valid {
		j.HasSalary = true
		j.SalaryMin = row.SalaryMin.Int32
		j.SalaryMax = row.SalaryMax.Int32
	}

	if row.DiscoveredAt.Valid {
		j.DiscoveredAt = row.DiscoveredAt.Time.Format("Jan 2")
	}

	for _, loc := range row.Locations {
		parts := strings.SplitN(loc, ":", 3)
		if len(parts) == 3 {
			j.Locations = append(j.Locations, templates.Location{
				Setting: parts[0],
				Country: parts[1],
				City:    parts[2],
			})
		}
	}

	j.Technologies = row.Technologies

	return j
}
