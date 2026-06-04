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

type place struct {
	Address struct {
		Locality string `json:"addressLocality"`
		Region   string `json:"addressRegion"`
		Country  string `json:"addressCountry"`
	} `json:"address"`
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
		sb.WriteString(formatLocations(d.Posting.JobLocation))
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

// formatLocations turns the jobLocation JSON-LD into "City, ST, Country" prose.
// The LLM normalizer reliably drops the city when handed the raw PostalAddress
// blob, so we hand it the shape the prompt expects. Falls back to the raw bytes
// if the structure is anything other than the Place(s) we know how to read.
func formatLocations(raw json.RawMessage) string {
	var places []place
	if err := json.Unmarshal(raw, &places); err != nil {
		var single place
		if err := json.Unmarshal(raw, &single); err != nil {
			return string(raw)
		}
		places = []place{single}
	}

	var lines []string
	for _, p := range places {
		var parts []string
		for _, f := range []string{p.Address.Locality, p.Address.Region, p.Address.Country} {
			if f != "" {
				parts = append(parts, f)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, strings.Join(parts, ", "))
		}
	}
	if len(lines) == 0 {
		return string(raw)
	}
	return strings.Join(lines, "; ")
}
