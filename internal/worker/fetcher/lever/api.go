package lever

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type LeverPosting struct {
	ID            string         `json:"id"`
	Text          string         `json:"text"`
	HostedURL     string         `json:"hostedUrl"`
	ApplyURL      string         `json:"applyUrl"`
	Country       string         `json:"country"`
	WorkplaceType string         `json:"workplaceType"`
	Categories    LeverCategories `json:"categories"`
	Description   string         `json:"description"`
	Additional    string         `json:"additional"`
	Lists         []LeverList    `json:"lists"`
}

type LeverCategories struct {
	Location     string   `json:"location"`
	Team         string   `json:"team"`
	Commitment   string   `json:"commitment"`
	Department   string   `json:"department"`
	AllLocations []string `json:"allLocations"`
}

type LeverList struct {
	Text    string `json:"text"`
	Content string `json:"content"`
}

func (p *LeverPosting) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(p)

	var sb strings.Builder
	sb.WriteString(p.Text)
	sb.WriteString("\n\n")
	if p.Categories.Location != "" {
		sb.WriteString("Location: ")
		sb.WriteString(p.Categories.Location)
		sb.WriteString("\n")
	}
	if len(p.Categories.AllLocations) > 0 {
		sb.WriteString("All Locations: ")
		sb.WriteString(strings.Join(p.Categories.AllLocations, ", "))
		sb.WriteString("\n")
	}
	if p.WorkplaceType != "" {
		sb.WriteString("Workplace: ")
		sb.WriteString(p.WorkplaceType)
		sb.WriteString("\n")
	}
	if p.Categories.Commitment != "" {
		sb.WriteString("Commitment: ")
		sb.WriteString(p.Categories.Commitment)
		sb.WriteString("\n")
	}
	if p.Categories.Team != "" {
		sb.WriteString("Team: ")
		sb.WriteString(p.Categories.Team)
		sb.WriteString("\n")
	}
	if p.Categories.Department != "" {
		sb.WriteString("Department: ")
		sb.WriteString(p.Categories.Department)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(p.Description)
	sb.WriteString("\n")
	for _, l := range p.Lists {
		if l.Text != "" {
			sb.WriteString(l.Text)
			sb.WriteString("\n")
		}
		sb.WriteString(l.Content)
		sb.WriteString("\n")
	}
	sb.WriteString(p.Additional)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: p.ID,
		URL:         p.HostedURL,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
