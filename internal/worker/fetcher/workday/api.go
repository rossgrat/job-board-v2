package workday

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type WorkdaySearchRequest struct {
	Limit         int            `json:"limit"`
	Offset        int            `json:"offset"`
	AppliedFacets map[string]any `json:"appliedFacets"`
	SearchText    string         `json:"searchText"`
}

type WorkdaySearchResponse struct {
	Total       int                 `json:"total"`
	JobPostings []WorkdayJobSummary `json:"jobPostings"`
}

type WorkdayJobSummary struct {
	Title         string   `json:"title"`
	ExternalPath  string   `json:"externalPath"`
	LocationsText string   `json:"locationsText"`
	PostedOn      string   `json:"postedOn"`
	BulletFields  []string `json:"bulletFields"`
}

type WorkdayJobDetail struct {
	JobPostingInfo WorkdayJobPostingInfo `json:"jobPostingInfo"`
}

type WorkdayJobPostingInfo struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	JobDescription string `json:"jobDescription"`
	Location       string `json:"location"`
	JobReqId       string `json:"jobReqId"`
	ExternalUrl    string `json:"externalUrl"`
	TimeType       string `json:"timeType"`
	StartDate      string `json:"startDate"`
	PostedOn       string `json:"postedOn"`
}

func (j *WorkdayJobPostingInfo) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(j)
	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: j.JobReqId,
		URL:         j.ExternalUrl,
		RawData:     rawData,
		CleanData:   model.CleanContent(rawData),
	}
}
