package filter

import (
	"encoding/json"
	"strings"

	"github.com/rossgrat/job-board-v2/database/gen/db"
)

func SplitConditions(conditions []db.FilterCondition) (locationConditions, jobConditions []db.FilterCondition) {
	for _, c := range conditions {
		if strings.HasPrefix(c.Field, "location_") {
			locationConditions = append(locationConditions, c)
		} else {
			jobConditions = append(jobConditions, c)
		}
	}
	return
}

func LocationPassesAll(location db.ClassifiedJobLocation, conditions []db.FilterCondition) bool {
	for _, c := range conditions {
		if !CheckCondition(ResolveLocationField(location, c.Field), c.Operator, c.Value) {
			return false
		}
	}
	return true
}

func ResolveLocationField(location db.ClassifiedJobLocation, field string) string {
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

func CheckCondition(fieldValue, operator, value string) bool {
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
