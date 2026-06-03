package icims

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
	ErrFetchList        = errors.New("failed to fetch list")
	ErrFetchDetail      = errors.New("failed to fetch detail")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrNoJobPosting     = errors.New("no JobPosting JSON-LD found")
	ErrMissingPageCount = errors.New("page count not found on iCIMS search page; layout changed")
)

type iCIMSConfig struct {
	Tenant string `json:"tenant"`
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

var (
	// iCIMS server-renders job links as /jobs/{id}/{slug}/job. The slug is
	// URL-encoded so it may contain %xx escapes; we capture both id and full
	// path to skip re-resolving the slug for detail fetches.
	detailLinkRe = regexp.MustCompile(`/jobs/(\d+)/[^"?]+/job`)

	// "Page N of M" is what the SubHeader shows; M tells us how many pages
	// to walk. If this disappears, layout changed.
	pageOfRe = regexp.MustCompile(`Page\s+\d+\s+of\s+(\d+)`)

	ldScriptRe = regexp.MustCompile(`(?s)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)

	// Each section of the job body (overview / responsibilities / qualifications)
	// is wrapped in a div with class iCIMS_InfoMsg_Job. We concatenate them all
	// for the CleanData feed.
	infoBlockRe = regexp.MustCompile(`(?s)<div class="iCIMS_InfoMsg iCIMS_InfoMsg_Job">(.*?)</div>\s*</div>\s*</div>`)
)

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "iCIMS::GetJobs"

	var cfg iCIMSConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}
	baseURL := fmt.Sprintf("https://%s.icims.com", cfg.Tenant)

	paths, err := c.listJobPaths(ctx, baseURL)
	if err != nil {
		return fmt.Errorf("%s:%w", fn, err)
	}

	var wg sync.WaitGroup
	for id, path := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			detail, err := c.fetchDetail(ctx, baseURL, id, path)
			if err != nil {
				slog.Error("failed to fetch icims detail",
					slog.String("tenant", cfg.Tenant),
					slog.String("id", id),
					slog.String("err", err.Error()))
				return
			}

			out <- detail.ToModel(companyID)
		}()
	}

	wg.Wait()
	return nil
}

// listJobPaths walks every paginated search results page and returns a map of
// {sourceJobID -> detail path}. The detail path embeds the slug iCIMS uses
// in the URL so we don't need to reconstruct it.
func (c *Client) listJobPaths(ctx context.Context, baseURL string) (map[string]string, error) {
	const fn = "iCIMS::listJobPaths"

	out := map[string]string{}

	firstBody, err := c.fetchSearchPage(ctx, baseURL, 0)
	if err != nil {
		return nil, fmt.Errorf("%s:%w", fn, err)
	}

	pageCountMatch := pageOfRe.FindSubmatch(firstBody)
	if pageCountMatch == nil {
		return nil, fmt.Errorf("%s:%w", fn, ErrMissingPageCount)
	}
	pageCount, _ := strconv.Atoi(string(pageCountMatch[1]))

	collectPaths(firstBody, out)

	for pr := 1; pr < pageCount; pr++ {
		body, err := c.fetchSearchPage(ctx, baseURL, pr)
		if err != nil {
			return nil, fmt.Errorf("%s:%w", fn, err)
		}
		collectPaths(body, out)
	}

	return out, nil
}

func collectPaths(body []byte, out map[string]string) {
	for _, m := range detailLinkRe.FindAllSubmatch(body, -1) {
		id := string(m[1])
		if _, ok := out[id]; ok {
			continue
		}
		out[id] = string(m[0])
	}
}

func (c *Client) fetchSearchPage(ctx context.Context, baseURL string, pr int) ([]byte, error) {
	const fn = "iCIMS::fetchSearchPage"

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchList, err)
	}

	url := fmt.Sprintf("%s/jobs/search?pr=%d&in_iframe=1", baseURL, pr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchList, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) fetchDetail(ctx context.Context, baseURL, id, path string) (*detailJob, error) {
	const fn = "iCIMS::fetchDetail"

	url := baseURL + path
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
		return nil, fmt.Errorf("%s:%w", fn, ErrNoJobPosting)
	}

	return &detailJob{
		ID:      id,
		URL:     url,
		Posting: posting,
		Body:    extractBody(body),
	}, nil
}

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

func extractBody(body []byte) string {
	matches := infoBlockRe.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return ""
	}
	var b []byte
	for i, m := range matches {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, m[1]...)
	}
	return string(b)
}
