package jibe

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type JibeListResponse struct {
	Jobs       []JibeJobEnvelope `json:"jobs"`
	TotalCount int               `json:"totalCount"`
}

type JibeJobEnvelope struct {
	Data JibeJob `json:"data"`
}

type JibeJob struct {
	Slug             string   `json:"slug"`
	ReqID            string   `json:"req_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Qualifications   string   `json:"qualifications"`
	Responsibilities string   `json:"responsibilities"`
	FullLocation     string   `json:"full_location"`
	City             string   `json:"city"`
	State            string   `json:"state"`
	Country          string   `json:"country"`
	EmploymentType   string         `json:"employment_type"`
	Categories       []JibeCategory `json:"categories"`
	ApplyURL         string         `json:"apply_url"`
}

type JibeCategory struct {
	Name string `json:"name"`
}

func (j *JibeJob) ToModel(companyID uuid.UUID, baseURL string) model.RawJob {
	rawData, _ := json.Marshal(j)

	var sb strings.Builder
	sb.WriteString(j.Title)
	sb.WriteString("\n\n")
	if j.FullLocation != "" {
		sb.WriteString("Location: ")
		sb.WriteString(j.FullLocation)
		sb.WriteString("\n")
	}
	if j.EmploymentType != "" {
		sb.WriteString("Employment: ")
		sb.WriteString(j.EmploymentType)
		sb.WriteString("\n")
	}
	if len(j.Categories) > 0 {
		names := make([]string, len(j.Categories))
		for i, c := range j.Categories {
			names[i] = c.Name
		}
		sb.WriteString("Categories: ")
		sb.WriteString(strings.Join(names, ", "))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(j.Description)
	sb.WriteString("\n")
	sb.WriteString(j.Responsibilities)
	sb.WriteString("\n")
	sb.WriteString(j.Qualifications)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: j.Slug,
		URL:         baseURL + "/jobs/" + j.Slug,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
