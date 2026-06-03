package smartrecruiters

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type SRListResponse struct {
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	TotalFound int        `json:"totalFound"`
	Content    []SRSummary `json:"content"`
}

type SRSummary struct {
	ID string `json:"id"`
}

type SRPosting struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	RefNumber  string     `json:"refNumber"`
	PostingURL string     `json:"postingUrl"`
	JobAd      SRJobAd    `json:"jobAd"`
}

type SRJobAd struct {
	Sections SRSections `json:"sections"`
}

type SRSections struct {
	CompanyDescription    SRSection `json:"companyDescription"`
	JobDescription        SRSection `json:"jobDescription"`
	Qualifications        SRSection `json:"qualifications"`
	AdditionalInformation SRSection `json:"additionalInformation"`
}

type SRSection struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (p *SRPosting) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(p)

	var sb strings.Builder
	sb.WriteString(p.Name)
	sb.WriteString("\n\n")
	for _, s := range []SRSection{
		p.JobAd.Sections.JobDescription,
		p.JobAd.Sections.Qualifications,
		p.JobAd.Sections.AdditionalInformation,
		p.JobAd.Sections.CompanyDescription,
	} {
		if s.Text == "" {
			continue
		}
		sb.WriteString(s.Title)
		sb.WriteString("\n")
		sb.WriteString(s.Text)
		sb.WriteString("\n\n")
	}

	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: p.ID,
		URL:         p.PostingURL,
		RawData:     rawData,
		CleanData:   model.CleanContent([]byte(sb.String())),
	}
}
