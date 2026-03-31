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
	DiscoveredAt   string
	UserStatus     string
	Locations      []Location
	Technologies   []string
}

type Location struct {
	Setting string
	Country string
	City    string
}

type CompanyItem struct {
	ID       string
	Name     string
	Favicon  string
	IsActive bool
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
	UserStatus   string
	CleanData    string
	RawData      string

	// Company fields
	CompanyName    string
	CompanyFavicon string
}
