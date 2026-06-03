package workable

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type WorkableSearchResponse struct {
	Total   int           `json:"total"`
	Results []WorkableJob `json:"results"`
}

type WorkableJob struct {
	ID         int               `json:"id"`
	Shortcode  string            `json:"shortcode"`
	Title      string            `json:"title"`
	Remote     bool              `json:"remote"`
	Location   WorkableLocation  `json:"location"`
	Locations  []WorkableLocation `json:"locations"`
	State      string            `json:"state"`
	Code       string            `json:"code"`
	Published  string            `json:"published"`
	Type       string            `json:"type"`
	Department []string          `json:"department"`
	Workplace  string            `json:"workplace"`

	// Description is fetched separately from the markdown view endpoint
	// and attached before serializing.
	Description string `json:"description,omitempty"`
}

type WorkableLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Region      *string `json:"region"`
}

func (wj *WorkableJob) URL() string {
	return fmt.Sprintf("https://apply.workable.com/j/%s", wj.Shortcode)
}

func (wj *WorkableJob) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(wj)
	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: wj.Shortcode,
		URL:         wj.URL(),
		RawData:     rawData,
		CleanData:   wj.Description,
	}
}
