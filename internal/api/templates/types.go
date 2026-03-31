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
