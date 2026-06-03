package doorsopen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
	"golang.org/x/time/rate"
)

var (
	ErrUnmarshalConfig  = errors.New("failed to unmarshal config")
	ErrCreateRequest    = errors.New("failed to create request")
	ErrFetchCompany     = errors.New("failed to fetch company page")
	ErrFetchDetail      = errors.New("failed to fetch detail")
	ErrUnexpectedStatus = errors.New("unexpected status code")

	// Sentinel parse errors. Each one means Doors Open has changed something
	// about their page structure and our scraper can no longer rely on the
	// layout. Returned as fetcher errors so the parent fetcher surfaces them
	// via telemetry (RecordFetchError) instead of silently logging zero jobs.
	ErrMissingTabPanel     = errors.New("all-vacancy tab panel not found on company page; page structure changed")
	ErrMissingJobCount     = errors.New("jobs count tab label not found on company page; page structure changed")
	ErrJobCountMismatch    = errors.New("jobs count tab label disagrees with extracted job ids; page structure changed")
	ErrNoJobPostingJSONLD  = errors.New("no JobPosting JSON-LD on detail page; page structure changed")
)

type DoorsOpenConfig struct {
	CompanyID string `json:"company_id"`
}

type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(5), 1),
	}
}

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "DoorsOpen::GetJobs"

	var cfg DoorsOpenConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	expectedCount, ids, err := c.listJobIDs(ctx, cfg.CompanyID)
	if err != nil {
		return fmt.Errorf("%s:%w", fn, err)
	}

	if expectedCount != len(ids) {
		// Sanity: the "Jobs (N)" tab label is what the company page advertises
		// as the open count. If we extract a different number from the listing
		// items, our regex missed something (or matched too much), and we'd
		// silently under- or over-fetch. Surface it.
		return fmt.Errorf("%s:%w: tab says %d, parsed %d", fn, ErrJobCountMismatch, expectedCount, len(ids))
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			detail, err := c.fetchDetail(ctx, id)
			if err != nil {
				slog.Error("failed to fetch doorsopen detail",
					slog.String("company_id", cfg.CompanyID),
					slog.String("job_id", id),
					slog.String("err", err.Error()))
				return
			}

			out <- detail.ToModel(companyID)
		}()
	}

	wg.Wait()
	return nil
}

var (
	// Match the "Jobs (N)" anchor in the company tab navigation. The tab
	// switcher lives in the same template across every Doors Open company
	// page, so this is our canary for layout changes.
	jobsCountRe = regexp.MustCompile(`(?s)aria-controls="all-vacancy"[^>]*>Jobs \((\d+)\)`)

	// Match /job/{id}/ permalinks inside the all-vacancy tab panel. The
	// panel starts at <div ... id="all-vacancy"> and ends at the next
	// sibling </div></div> wrapping. We scope by extracting the panel
	// substring first.
	tabPanelRe   = regexp.MustCompile(`(?s)<div[^>]*id="all-vacancy"[^>]*>(.*)`)
	detailLinkRe = regexp.MustCompile(`/job/(\d+)/`)
)

// listJobIDs fetches the company page, asserts the layout markers we depend
// on are still there, and returns (expectedCount from the tab label, unique
// job IDs from the all-vacancy panel).
func (c *Client) listJobIDs(ctx context.Context, doorsCompanyID string) (int, []string, error) {
	const fn = "DoorsOpen::listJobIDs"

	if err := c.limiter.Wait(ctx); err != nil {
		return 0, nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchCompany, err)
	}

	url := fmt.Sprintf("https://www.doorsopen.co/company/%s/", doorsCompanyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchCompany, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchCompany, err)
	}

	return parseCompanyPage(body)
}

// parseCompanyPage extracts the advertised "Jobs (N)" count and the unique
// /job/{id}/ IDs scoped to the all-vacancy tab panel. Either step missing is
// a structural-change signal and returned as an error rather than swallowed.
func parseCompanyPage(body []byte) (int, []string, error) {
	countMatch := jobsCountRe.FindSubmatch(body)
	if countMatch == nil {
		return 0, nil, ErrMissingJobCount
	}
	expectedCount, _ := strconv.Atoi(string(countMatch[1]))

	panelMatch := tabPanelRe.FindSubmatch(body)
	if panelMatch == nil {
		return 0, nil, ErrMissingTabPanel
	}
	panel := panelMatch[1]

	seen := map[string]struct{}{}
	var ids []string
	for _, m := range detailLinkRe.FindAllSubmatch(panel, -1) {
		id := string(m[1])
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return expectedCount, ids, nil
}

func (c *Client) fetchDetail(ctx context.Context, id string) (*detailJob, error) {
	const fn = "DoorsOpen::fetchDetail"

	url := fmt.Sprintf("https://www.doorsopen.co/job/%s/", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchDetail, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchDetail, err)
	}

	posting, ok := extractJobPosting(body)
	if !ok {
		return nil, fmt.Errorf("%s:%w", fn, ErrNoJobPostingJSONLD)
	}

	return &detailJob{
		ID:      id,
		URL:     url,
		Posting: posting,
	}, nil
}

// ldScriptRe captures every JSON-LD <script> block; we pick out the
// JobPosting one (the page also ships BreadcrumbList, WebSite, etc).
var ldScriptRe = regexp.MustCompile(`(?s)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)

func extractJobPosting(body []byte) (JobPosting, bool) {
	for _, m := range ldScriptRe.FindAllSubmatch(body, -1) {
		var probe struct {
			Type string `json:"@type"`
		}
		if err := json.Unmarshal(m[1], &probe); err != nil {
			continue
		}
		if probe.Type != "JobPosting" {
			continue
		}
		var posting JobPosting
		if err := json.Unmarshal(m[1], &posting); err != nil {
			continue
		}
		return posting, true
	}
	return JobPosting{}, false
}
