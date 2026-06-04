package api

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	queries := db.New(s.pool)

	rows, err := queries.ListCompaniesWithJobCounts(r.Context())
	if err != nil {
		slog.Error("failed to list company metrics", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var total templates.CompanyJobCounts
	companies := make([]templates.MetricsCompany, 0, len(rows))
	for _, row := range rows {
		counts := templates.CompanyJobCounts{
			Total:             row.Total,
			Technical:         row.Technical,
			Accepted:          row.Accepted,
			FilteredRelevance: row.FilteredRelevance,
			NonTechnical:      row.NonTechnical,
			Pending:           row.Pending,
			Dead:              row.Dead,
		}
		total.Total += counts.Total
		total.Technical += counts.Technical
		total.Accepted += counts.Accepted
		total.FilteredRelevance += counts.FilteredRelevance
		total.NonTechnical += counts.NonTechnical
		total.Pending += counts.Pending
		total.Dead += counts.Dead

		companies = append(companies, templates.MetricsCompany{
			Name:    row.Name,
			Favicon: row.FaviconUrl,
			Counts:  counts,
		})
	}

	sort.SliceStable(companies, func(i, j int) bool {
		return companies[i].Counts.Total > companies[j].Counts.Total
	})

	templates.MetricsPage(total, companies).Render(r.Context(), w)
}
