package atomfeed

import (
	"encoding/json"
	"encoding/xml"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	Title   string   `xml:"title" json:"title"`
	ID      string   `xml:"id" json:"id"`
	Link    AtomLink `xml:"link" json:"link"`
	Updated string   `xml:"updated" json:"updated"`
	Content string   `xml:"content" json:"content"`
}

type AtomLink struct {
	Href string `xml:"href,attr" json:"href"`
}

func (e *AtomEntry) ToModel(companyID uuid.UUID) model.RawJob {
	rawData, _ := json.Marshal(e)
	return model.RawJob{
		CompanyID:   companyID,
		SourceJobID: e.ID + "|" + e.Updated,
		URL:         e.Link.Href,
		RawData:     rawData,
		CleanData:   model.CleanContent(rawData),
	}
}
