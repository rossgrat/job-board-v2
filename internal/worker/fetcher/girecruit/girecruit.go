package girecruit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
	"golang.org/x/time/rate"
)

var (
	ErrUnmarshalConfig  = errors.New("failed to unmarshal config")
	ErrCreateRequest    = errors.New("failed to create request")
	ErrFetchIndex       = errors.New("failed to fetch index")
	ErrFetchDetail      = errors.New("failed to fetch detail")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrNoJobPosting     = errors.New("no JobPosting JSON-LD found")
)

type GiRecruitConfig struct {
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

var detailLinkRe = regexp.MustCompile(`/job/detail/(\d+)`)

// ldScriptRe matches a non-greedy JSON-LD <script> block. Attributes can
// appear in any order, so the regex doesn't anchor on "type" being first.
var ldScriptRe = regexp.MustCompile(`(?s)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "GiRecruit::GetJobs"

	var cfg GiRecruitConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}
	baseURL := fmt.Sprintf("https://%s.gi-recruit.com", cfg.Tenant)

	ids, err := c.listJobIDs(ctx, baseURL)
	if err != nil {
		return fmt.Errorf("%s:%w", fn, err)
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			detail, err := c.fetchDetail(ctx, baseURL, id)
			if err != nil {
				slog.Error("failed to fetch gi-recruit detail",
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

func (c *Client) listJobIDs(ctx context.Context, baseURL string) ([]string, error) {
	const fn = "GiRecruit::listJobIDs"

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchIndex, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchIndex, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchIndex, err)
	}

	seen := map[string]struct{}{}
	var ids []string
	for _, m := range detailLinkRe.FindAllStringSubmatch(string(body), -1) {
		id := m[1]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *Client) fetchDetail(ctx context.Context, baseURL, id string) (*detailJob, error) {
	const fn = "GiRecruit::fetchDetail"

	url := fmt.Sprintf("%s/job/detail/%s", baseURL, id)
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
	}, nil
}

// extractJobPosting walks every JSON-LD script block on the page and returns
// the first one whose @type is "JobPosting". gi-recruit pages contain a
// BreadcrumbList block in addition to the JobPosting block, so a naive
// "first match" approach picks the wrong one.
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
