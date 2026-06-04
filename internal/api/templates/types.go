package templates

type DashboardJob struct {
	ID             string
	Title          string
	URL            string
	CompanyName    string
	CompanyFavicon string
	Level          string
	SalaryMin      int32
	SalaryMax      int32
	HasSalary      bool
	Category       string
	Relevance      string
	Status         string
	DiscoveredAt   string
	UserStatus     string
	Locations      []Location
	Technologies   []string
	HasEval        bool
}

type Location struct {
	Setting string
	Country string
	City    string
}

type CompanyItem struct {
	ID        string
	Name      string
	Favicon   string
	FetchType string
	IsActive  bool
}

type CompanyJobCounts struct {
	Total             int64
	Technical         int64
	Accepted          int64
	FilteredRelevance int64
	NonTechnical      int64
	Pending           int64
	Dead              int64
}

type MetricsCompany struct {
	Name    string
	Favicon string
	Counts  CompanyJobCounts
}

type CountSegment struct {
	Label string
	Class string
	Count int64
	Pct   float64
}

// Segments returns the mutually-exclusive status buckets that sum to Total,
// each with its share of the bar width.
func (n CompanyJobCounts) Segments() []CountSegment {
	segs := []CountSegment{
		{Label: "accepted", Class: "seg--accepted", Count: n.Accepted},
		{Label: "filtered", Class: "seg--filtered", Count: n.FilteredRelevance},
		{Label: "non-technical", Class: "seg--non-technical", Count: n.NonTechnical},
		{Label: "pending", Class: "seg--pending", Count: n.Pending},
		{Label: "dead", Class: "seg--dead", Count: n.Dead},
	}
	if n.Total > 0 {
		for i := range segs {
			segs[i].Pct = float64(segs[i].Count) / float64(n.Total) * 100
		}
	}
	return segs
}

type FilterState struct {
	Status              string
	Relevance           string
	UserStatus          string
	CompanyName         string
	CompanyNames        []string
	IncludeNonTechnical bool
}

type ReviewModal struct {
	ClassifiedJobID  string
	RawJobID         string
	Title            string
	UserStatus       string
	RejectionReason  string
	ModelCategory    string
	ModelRelevance   string
	EvalCategory     string
	EvalRelevance    string
	HasEval          bool
}

type JobDetail struct {
	// Classified job fields
	ID                          string
	Status                      string
	IsCurrent                   bool
	Title                       string
	Level                       string
	SalaryMin                   int32
	SalaryMax                   int32
	HasSalary                   bool
	Category                    string
	Relevance                   string
	Reasoning                   string
	ClassificationPromptVersion string
	CreatedAt                   string
	NormalizedAt                string
	ClassifiedAt                string
	Locations                   []Location
	Technologies                []string

	// Raw job fields
	RawJobID     string
	URL          string
	SourceJobID  string
	DiscoveredAt string
	UserStatus      string
	RejectionReason string
	CleanData       string
	RawData         string

	// Company fields
	CompanyName    string
	CompanyFavicon string
}
