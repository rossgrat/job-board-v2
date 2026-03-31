package api

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
	"github.com/rossgrat/job-board-v2/internal/filter"
)

// locationFilterGroup holds the location-only conditions for a single filter group.
type locationFilterGroup []db.FilterCondition

func (s *Server) loadLocationFilters(ctx context.Context) ([]locationFilterGroup, error) {
	queries := db.New(s.pool)

	groups, err := queries.GetActiveFilterGroups(ctx)
	if err != nil {
		return nil, err
	}

	var filters []locationFilterGroup
	for _, g := range groups {
		conditions, err := queries.GetFilterConditionsByGroupID(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		locConditions, _ := filter.SplitConditions(conditions)
		if len(locConditions) > 0 {
			filters = append(filters, locConditions)
		}
	}
	return filters, nil
}

func filterLocations(locations []templates.Location, groups []locationFilterGroup) []templates.Location {
	if len(groups) == 0 {
		return locations
	}

	var filtered []templates.Location
	for _, loc := range locations {
		dbLoc := db.ClassifiedJobLocation{
			Setting: loc.Setting,
			Country: loc.Country,
			City:    pgtype.Text{String: loc.City, Valid: loc.City != ""},
		}
		for _, g := range groups {
			if filter.LocationPassesAll(dbLoc, g) {
				filtered = append(filtered, loc)
				break
			}
		}
	}

	if len(filtered) == 0 {
		return locations
	}
	return filtered
}
