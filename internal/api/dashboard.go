package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	queries := db.New(s.pool)
	ctx := r.Context()

	rows, err := queries.ListDashboardJobs(ctx)
	if err != nil {
		slog.Error("failed to list dashboard jobs", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	locFilters, err := s.loadLocationFilters(ctx)
	if err != nil {
		slog.Error("failed to load location filters", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var newJobs, tabledJobs []templates.DashboardJob
	for _, row := range rows {
		j := toDashboardJob(row)
		j.Locations = filterLocations(j.Locations, locFilters)
		if j.UserStatus == "tabled" {
			tabledJobs = append(tabledJobs, j)
		} else {
			newJobs = append(newJobs, j)
		}
	}

	templates.DashboardPage(newJobs, tabledJobs).Render(ctx, w)
}

func toDashboardJob(row db.ListDashboardJobsRow) templates.DashboardJob {
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
	j.HasEval = row.HasEval

	return j
}
