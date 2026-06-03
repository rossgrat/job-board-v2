package pinpoint

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type PinpointListResponse struct {
	Data []PinpointPosting `json:"data"`
}

type PinpointPosting struct {
	ID                       string           `json:"id"`
	Title                    string           `json:"title"`
	URL                      string           `json:"url"`
	Path                     string           `json:"path"`
	Description              string           `json:"description"`
	KeyResponsibilities      string           `json:"key_responsibilities"`
	SkillsKnowledgeExpertise string           `json:"skills_knowledge_expertise"`
	Benefits                 string           `json:"benefits"`
	EmploymentTypeText       string           `json:"employment_type_text"`
	WorkplaceTypeText        string           `json:"workplace_type_text"`
	Job                      PinpointJobMeta  `json:"job"`
	Location                 PinpointLocation `json:"location"`
}

type PinpointJobMeta struct {
	Department PinpointNamed `json:"department"`
	Division   PinpointNamed `json:"division"`
}

type PinpointNamed struct {
	Name string `json:"name"`
}

type PinpointLocation struct {
	Name       string `json:"name"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postal_code"`
}

func (p *PinpointPosting) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(p)

	var sb strings.Builder
	sb.WriteString(p.Title)
	sb.WriteString("\n\n")
	if p.WorkplaceTypeText != "" {
		sb.WriteString("Workplace: ")
		sb.WriteString(p.WorkplaceTypeText)
		sb.WriteString("\n")
	}
	if p.EmploymentTypeText != "" {
		sb.WriteString("Employment: ")
		sb.WriteString(p.EmploymentTypeText)
		sb.WriteString("\n")
	}
	if p.Location.Name != "" {
		sb.WriteString("Location: ")
		sb.WriteString(p.Location.Name)
		sb.WriteString("\n")
	}
	if p.Job.Department.Name != "" {
		sb.WriteString("Department: ")
		sb.WriteString(p.Job.Department.Name)
		sb.WriteString("\n")
	}
	if p.Job.Division.Name != "" {
		sb.WriteString("Division: ")
		sb.WriteString(p.Job.Division.Name)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(p.Description)
	sb.WriteString("\n")
	sb.WriteString(p.KeyResponsibilities)
	sb.WriteString("\n")
	sb.WriteString(p.SkillsKnowledgeExpertise)
	sb.WriteString("\n")
	sb.WriteString(p.Benefits)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: p.ID,
		URL:         p.URL,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
