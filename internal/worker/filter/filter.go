package filter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/outbox"
)

var (
	ErrLoadClassifiedJob = errors.New("failed to load classified job")
	ErrLoadLocations     = errors.New("failed to load locations")
	ErrLoadFilterGroups  = errors.New("failed to load filter groups")
	ErrLoadConditions    = errors.New("failed to load filter conditions")
	ErrUpdateStatus      = errors.New("failed to update classified job status")
)

// filterResult ranks how far a job got through a filter group's evaluation.
// Higher values mean the job passed more conditions (closer miss).
type filterResult int

const (
	filterNone       filterResult = iota
	filteredLocation
	filteredLevel
)

var resultToStatus = map[filterResult]string{
	filteredLocation: constants.StatusFilteredLocation,
	filteredLevel:    constants.StatusFilteredLevel,
}

var fieldToResult = map[string]filterResult{
	"level": filteredLevel,
}

type Handler struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Handle(ctx context.Context, task db.OutboxTask) (*outbox.TaskResult, error) {
	const fn = "FilterHandler::Handle"

	queries := db.New(h.pool)

	job, err := queries.GetClassifiedJobByID(ctx, task.ClassifiedJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadClassifiedJob, err)
	}

	locations, err := queries.GetClassifiedJobLocations(ctx, task.ClassifiedJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadLocations, err)
	}

	groups, err := queries.GetActiveFilterGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadFilterGroups, err)
	}

	var bestResult filterResult
	for _, group := range groups {
		conditions, err := queries.GetFilterConditionsByGroupID(ctx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadConditions, err)
		}

		passed, result := evaluateGroup(job, locations, conditions)
		if passed {
			return &outbox.TaskResult{NextTaskName: constants.PipelineClassify}, nil
		}
		if result > bestResult {
			bestResult = result
		}
	}

	err = queries.UpdateClassifiedJobStatus(ctx, db.UpdateClassifiedJobStatusParams{
		ID:     task.ClassifiedJobID,
		Status: resultToStatus[bestResult],
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateStatus, err)
	}

	return &outbox.TaskResult{}, nil
}

func evaluateGroup(job db.ClassifiedJob, locations []db.ClassifiedJobLocation, conditions []db.FilterCondition) (bool, filterResult) {
	locationConditions, jobConditions := splitConditions(conditions)

	if len(locationConditions) > 0 && !anyLocationPassesAll(locations, locationConditions) {
		return false, filteredLocation
	}

	for _, condition := range jobConditions {
		if !checkCondition(resolveJobField(job, condition.Field), condition.Operator, condition.Value) {
			return false, fieldToResult[condition.Field]
		}
	}

	return true, filterNone
}

func splitConditions(conditions []db.FilterCondition) ([]db.FilterCondition, []db.FilterCondition) {
	var locationConditions, jobConditions []db.FilterCondition
	for _, condition := range conditions {
		if strings.HasPrefix(condition.Field, "location_") {
			locationConditions = append(locationConditions, condition)
		} else {
			jobConditions = append(jobConditions, condition)
		}
	}
	return locationConditions, jobConditions
}

func anyLocationPassesAll(locations []db.ClassifiedJobLocation, conditions []db.FilterCondition) bool {
	for _, location := range locations {
		if locationPassesAll(location, conditions) {
			return true
		}
	}
	return false
}

func locationPassesAll(location db.ClassifiedJobLocation, conditions []db.FilterCondition) bool {
	for _, condition := range conditions {
		if !checkCondition(resolveLocationField(location, condition.Field), condition.Operator, condition.Value) {
			return false
		}
	}
	return true
}

func resolveJobField(job db.ClassifiedJob, field string) string {
	switch field {
	case "level":
		return job.Level.String
	default:
		return ""
	}
}

func resolveLocationField(location db.ClassifiedJobLocation, field string) string {
	switch field {
	case "location_country":
		return location.Country
	case "location_city":
		return location.City.String
	case "location_setting":
		return location.Setting
	default:
		return ""
	}
}

func checkCondition(fieldValue, operator, value string) bool {
	switch operator {
	case "equals":
		return strings.EqualFold(fieldValue, value)
	case "not_equals":
		return !strings.EqualFold(fieldValue, value)
	case "contains":
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value))
	case "in":
		return isInList(fieldValue, value)
	case "not_in":
		return !isInList(fieldValue, value)
	default:
		return false
	}
}

func isInList(fieldValue, jsonArray string) bool {
	var list []string
	if err := json.Unmarshal([]byte(jsonArray), &list); err != nil {
		return false
	}
	for _, item := range list {
		if strings.EqualFold(fieldValue, item) {
			return true
		}
	}
	return false
}
