package workday

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
	"golang.org/x/time/rate"
)

var (
	ErrUnmarshalConfig  = errors.New("failed to unmarshal config")
	ErrCreateRequest    = errors.New("failed to create request")
	ErrFetchPage        = errors.New("failed to fetch page")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrDecodePage       = errors.New("failed to decode page")
	ErrFetchDetail      = errors.New("failed to fetch detail")
	ErrDecodeDetail     = errors.New("failed to decode detail")
)

type WorkdayConfig struct {
	BaseURL string `json:"base_url"`
	Tenant  string `json:"tenant"`
	Site    string `json:"site"`
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
	const fn = "Workday::GetJobs"

	var cfg WorkdayConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}
	baseAPI := fmt.Sprintf("%s/wday/cxs/%s/%s", cfg.BaseURL, cfg.Tenant, cfg.Site)

	summaries := make(chan WorkdayJobSummary)
	producerErr := make(chan error, 1)
	go c.produceSummaries(ctx, baseAPI, summaries, producerErr)

	c.consumeDetails(ctx, baseAPI, companyID, summaries, out)

	if err := <-producerErr; err != nil {
		return err
	}

	return nil
}

// produceSummaries paginates through the list endpoint and sends each job
// summary into the out channel. Closes the channel when done.
func (c *Client) produceSummaries(ctx context.Context, baseAPI string, out chan<- WorkdayJobSummary, errCh chan<- error) {
	const fn = "Workday::produceSummaries"
	defer close(out)

	const pageSize = 20
	url := baseAPI + "/jobs"

	total := 0
	for offset := 0; ; offset += pageSize {
		if err := c.limiter.Wait(ctx); err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrFetchPage, err)
			return
		}

		body, _ := json.Marshal(WorkdaySearchRequest{
			Limit:         pageSize,
			Offset:        offset,
			AppliedFacets: map[string]any{},
		})

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrFetchPage, err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
			return
		}

		var page WorkdaySearchResponse
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrDecodePage, err)
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
// concurrently, sending each result to the output channel.
func (c *Client) consumeDetails(ctx context.Context, baseAPI string, companyID uuid.UUID, summaries <-chan WorkdayJobSummary, out chan<- model.RawJob) {
	var wg sync.WaitGroup

	for summary := range summaries {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			detail, err := c.fetchDetail(ctx, baseAPI, summary.ExternalPath)
			if err != nil {
				slog.Error("failed to fetch workday job detail",
					slog.String("path", summary.ExternalPath),
					slog.String("err", err.Error()))
				return
			}

			out <- detail.JobPostingInfo.ToModel(companyID)
		}()
	}

	wg.Wait()
}

func (c *Client) fetchDetail(ctx context.Context, baseAPI string, externalPath string) (*WorkdayJobDetail, error) {
	const fn = "Workday::fetchDetail"

	url := fmt.Sprintf("%s%s", baseAPI, externalPath)

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

	var detail WorkdayJobDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeDetail, err)
	}

	return &detail, nil
}
