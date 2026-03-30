package greenhouse

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type GreenhouseJobsResponse struct {
	Jobs []GreenhouseJob `json:"jobs"`
}

type GreenhouseJob struct {
	ID             int                    `json:"id"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	AbsoluteURL    string                 `json:"absolute_url"`
	Location       GreenhouseLocation     `json:"location"`
	UpdatedAt      string                 `json:"updated_at"`
	FirstPublished string                 `json:"first_published"`
	CompanyName    string                 `json:"company_name"`
	Departments    []GreenhouseDepartment `json:"departments"`
	Offices        []GreenhouseOffice     `json:"offices"`
}

func (gj *GreenhouseJob) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(gj)
	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: strconv.Itoa(gj.ID),
		URL:         gj.AbsoluteURL,
		RawData:     rawData,
		CleanData:   model.CleanContent(rawData),
	}
}

type GreenhouseLocation struct {
	Name string `json:"name"`
}

type GreenhouseDepartment struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GreenhouseOffice struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}
