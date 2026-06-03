package girecruit

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type JobPosting struct {
	Context         string          `json:"@context"`
	Type            string          `json:"@type"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	DatePosted      string          `json:"datePosted"`
	ValidThrough    string          `json:"validThrough"`
	EmploymentType  string          `json:"employmentType"`
	BaseSalary      json.RawMessage `json:"baseSalary,omitempty"`
	JobLocation     json.RawMessage `json:"jobLocation,omitempty"`
	HiringOrg       json.RawMessage `json:"hiringOrganization,omitempty"`
	Identifier      json.RawMessage `json:"identifier,omitempty"`
}

// detailJob carries the JSON-LD payload plus the gi-recruit ID we used to
// locate it. The ID is the source job ID we persist.
type detailJob struct {
	ID      string
	URL     string
	Posting JobPosting
}

func (d *detailJob) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(d.Posting)
	clean := model.CleanContent([]byte(d.Posting.Title + "\n\n" + d.Posting.Description))
	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: d.ID,
		URL:         d.URL,
		RawData:     rawData,
		CleanData:   clean,
	}
}
