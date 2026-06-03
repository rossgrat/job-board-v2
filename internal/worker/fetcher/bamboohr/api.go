package bamboohr

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type BambooHRListResponse struct {
	Meta   BambooHRListMeta `json:"meta"`
	Result []BambooHRJob    `json:"result"`
}

type BambooHRListMeta struct {
	TotalCount int `json:"totalCount"`
}

type BambooHRJob struct {
	ID              json.Number `json:"id"`
	JobOpeningName  string      `json:"jobOpeningName"`
	DepartmentLabel string      `json:"departmentLabel"`
}

type BambooHRDetailResponse struct {
	Result struct {
		JobOpening BambooHRJobOpening `json:"jobOpening"`
	} `json:"result"`
}

type BambooHRJobOpening struct {
	JobOpeningShareUrl    string `json:"jobOpeningShareUrl"`
	JobOpeningName        string `json:"jobOpeningName"`
	JobOpeningStatus      string `json:"jobOpeningStatus"`
	DepartmentLabel       string `json:"departmentLabel"`
	EmploymentStatusLabel string `json:"employmentStatusLabel"`
	Description           string `json:"description"`
}

func (j *BambooHRJobOpening) ToModel(companyID uuid.UUID, id string) model.RawJob {
	rawData, _ := json.Marshal(j)

	var sb strings.Builder
	sb.WriteString(j.JobOpeningName)
	sb.WriteString("\n\n")
	if j.DepartmentLabel != "" {
		sb.WriteString("Department: ")
		sb.WriteString(j.DepartmentLabel)
		sb.WriteString("\n")
	}
	if j.EmploymentStatusLabel != "" {
		sb.WriteString("Employment: ")
		sb.WriteString(j.EmploymentStatusLabel)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(j.Description)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: id,
		URL:         j.JobOpeningShareUrl,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
