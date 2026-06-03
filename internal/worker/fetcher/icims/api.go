package icims

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type JobPosting struct {
	Context            string          `json:"@context"`
	Type               string          `json:"@type"`
	Title              string          `json:"title"`
	DatePosted         string          `json:"datePosted"`
	ValidThrough       string          `json:"validThrough"`
	EmploymentType     string          `json:"employmentType"`
	OccupationalCat    string          `json:"occupationalCategory"`
	URL                string          `json:"url"`
	HiringOrganization json.RawMessage `json:"hiringOrganization,omitempty"`
	JobLocation        json.RawMessage `json:"jobLocation,omitempty"`
}

// detailJob holds everything we scraped from a single job detail page.
// The JSON-LD block carries metadata but not the body copy, so we also keep
// the stitched-together iCIMS_InfoMsg_Job text for the LLM to read.
type detailJob struct {
	ID      string
	URL     string
	Posting JobPosting
	Body    string
}

func (d *detailJob) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(d.Posting)

	var sb strings.Builder
	sb.WriteString(d.Posting.Title)
	sb.WriteString("\n\n")
	if d.Posting.OccupationalCat != "" {
		sb.WriteString("Category: ")
		sb.WriteString(d.Posting.OccupationalCat)
		sb.WriteString("\n")
	}
	if d.Posting.EmploymentType != "" {
		sb.WriteString("Employment: ")
		sb.WriteString(d.Posting.EmploymentType)
		sb.WriteString("\n")
	}
	if len(d.Posting.JobLocation) > 0 {
		sb.WriteString("Location: ")
		sb.Write(d.Posting.JobLocation)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(d.Body)

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: d.ID,
		URL:         d.URL,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
