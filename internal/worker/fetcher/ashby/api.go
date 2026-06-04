package ashby

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type AshbyListResponse struct {
	Jobs []AshbyJob `json:"jobs"`
}

type AshbyJob struct {
	ID                string                  `json:"id"`
	Title             string                  `json:"title"`
	Department        string                  `json:"department"`
	Team              string                  `json:"team"`
	EmploymentType    string                  `json:"employmentType"`
	Location          string                  `json:"location"`
	SecondaryLocations []AshbySecondaryLocation `json:"secondaryLocations"`
	IsRemote          bool                    `json:"isRemote"`
	WorkplaceType     string                  `json:"workplaceType"`
	JobURL            string                  `json:"jobUrl"`
	ApplyURL          string                  `json:"applyUrl"`
	DescriptionHTML   string                  `json:"descriptionHtml"`
}

type AshbySecondaryLocation struct {
	Location string `json:"location"`
}

func (j *AshbyJob) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(j)

	var sb strings.Builder
	sb.WriteString(j.Title)
	sb.WriteString("\n\n")
	if j.Location != "" {
		sb.WriteString("Location: ")
		sb.WriteString(j.Location)
		sb.WriteString("\n")
	}
	if len(j.SecondaryLocations) > 0 {
		var locs []string
		for _, l := range j.SecondaryLocations {
			if l.Location != "" {
				locs = append(locs, l.Location)
			}
		}
		if len(locs) > 0 {
			sb.WriteString("Secondary Locations: ")
			sb.WriteString(strings.Join(locs, ", "))
			sb.WriteString("\n")
		}
	}
	if j.WorkplaceType != "" {
		sb.WriteString("Workplace: ")
		sb.WriteString(j.WorkplaceType)
		sb.WriteString("\n")
	}
	if j.EmploymentType != "" {
		sb.WriteString("Employment: ")
		sb.WriteString(j.EmploymentType)
		sb.WriteString("\n")
	}
	if j.Department != "" {
		sb.WriteString("Department: ")
		sb.WriteString(j.Department)
		sb.WriteString("\n")
	}
	if j.Team != "" {
		sb.WriteString("Team: ")
		sb.WriteString(j.Team)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(j.DescriptionHTML)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: j.ID,
		URL:         j.JobURL,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
