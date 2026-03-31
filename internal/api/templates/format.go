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
