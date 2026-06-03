package templates

import "fmt"

func formatSalary(min, max int32) string {
	return fmt.Sprintf("$%dk–$%dk", min/1000, max/1000)
}

func formatBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func statusVal(status string) string {
	return fmt.Sprintf(`{"status":"%s"}`, status)
}

// hasRelevance reports whether jobs at the given pipeline status have a
// non-null relevance score. Classify runs after the hard filter, so jobs
// rejected at the hard filter (filtered_location, filtered_level) never get
// scored. The browse view uses this to hide the relevance and user_status
// dropdowns when filtering to those buckets, since they wouldn't match.
func hasRelevance(status string) bool {
	switch status {
	case "filtered_location", "filtered_level":
		return false
	}
	return true
}
