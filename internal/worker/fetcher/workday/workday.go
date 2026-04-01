package workday

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type WorkdayConfig struct {
	BaseURL string `json:"base_url"`
	Tenant  string `json:"tenant"`
	Site    string `json:"site"`
}

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte) ([]model.RawJob, error) {
	var cfg WorkdayConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling workday config: %w", err)
	}
	baseAPI := fmt.Sprintf("%s/wday/cxs/%s/%s", cfg.BaseURL, cfg.Tenant, cfg.Site)

	summaries := make(chan WorkdayJobSummary)
	producerErr := make(chan error, 1)
	go c.produceSummaries(ctx, baseAPI, summaries, producerErr)

	rawJobs := c.consumeDetails(ctx, baseAPI, companyID, summaries)

	if err := <-producerErr; err != nil {
		return nil, err
	}

	return rawJobs, nil
}

// produceSummaries paginates through the list endpoint and sends each job
// summary into the out channel. Closes the channel when done.
func (c *Client) produceSummaries(ctx context.Context, baseAPI string, out chan<- WorkdayJobSummary, errCh chan<- error) {
	defer close(out)

	const pageSize = 20
	url := baseAPI + "/jobs"

	total := 0
	for offset := 0; ; offset += pageSize {
		body, _ := json.Marshal(WorkdaySearchRequest{
			Limit:         pageSize,
			Offset:        offset,
			AppliedFacets: map[string]any{},
		})

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("creating list request: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("fetching page at offset %d: %w", offset, err)
			return
		}

		var page WorkdaySearchResponse
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			errCh <- fmt.Errorf("decoding page at offset %d: %w", offset, err)
			return
		}

		if offset == 0 {
			total = page.Total
		}

		for _, s := range page.JobPostings {
			out <- s
		}

		if offset+pageSize >= total {
			break
		}
	}

	errCh <- nil
}

// consumeDetails reads summaries from the channel and fetches full job details
// concurrently, limited to 5 in-flight requests.
func (c *Client) consumeDetails(ctx context.Context, baseAPI string, companyID uuid.UUID, summaries <-chan WorkdayJobSummary) []model.RawJob {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 5)
		rawJobs []model.RawJob
	)

	for summary := range summaries {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			detail, err := c.fetchDetail(ctx, baseAPI, summary.ExternalPath)
			if err != nil {
				slog.Error("failed to fetch workday job detail",
					slog.String("path", summary.ExternalPath),
					slog.String("err", err.Error()))
				return
			}

			job := detail.JobPostingInfo.ToModel(companyID)
			mu.Lock()
			rawJobs = append(rawJobs, job)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return rawJobs
}

func (c *Client) fetchDetail(ctx context.Context, baseAPI string, externalPath string) (*WorkdayJobDetail, error) {
	url := fmt.Sprintf("%s%s", baseAPI, externalPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching detail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var detail WorkdayJobDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("decoding detail: %w", err)
	}

	return &detail, nil
}
