package smartrecruiters

import (
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
	ErrDecodeResponse   = errors.New("failed to decode response")
	ErrFetchDetail      = errors.New("failed to fetch detail")
)

type SmartRecruitersConfig struct {
	CompanyIdentifier string `json:"company_identifier"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New() *Client {
	return &Client{
		baseURL:    "https://api.smartrecruiters.com/v1/companies",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(5), 1),
	}
}

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "SmartRecruiters::GetJobs"

	var cfg SmartRecruitersConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	summaries := make(chan SRSummary)
	producerErr := make(chan error, 1)
	go c.produceSummaries(ctx, cfg.CompanyIdentifier, summaries, producerErr)

	c.consumeDetails(ctx, cfg.CompanyIdentifier, companyID, summaries, out)

	return <-producerErr
}

func (c *Client) produceSummaries(ctx context.Context, slug string, out chan<- SRSummary, errCh chan<- error) {
	const fn = "SmartRecruiters::produceSummaries"
	defer close(out)

	const pageSize = 100
	total := 0
	for offset := 0; ; offset += pageSize {
		if err := c.limiter.Wait(ctx); err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrFetchPage, err)
			return
		}

		url := fmt.Sprintf("%s/%s/postings?limit=%d&offset=%d", c.baseURL, slug, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
			return
		}

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

		var page SRListResponse
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			errCh <- fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
			return
		}

		if offset == 0 {
			total = page.TotalFound
		}

		for _, s := range page.Content {
			out <- s
		}

		if offset+pageSize >= total {
			break
		}
	}

	errCh <- nil
}

func (c *Client) consumeDetails(ctx context.Context, slug string, companyID uuid.UUID, summaries <-chan SRSummary, out chan<- model.RawJob) {
	var wg sync.WaitGroup

	for summary := range summaries {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			posting, err := c.fetchPosting(ctx, slug, summary.ID)
			if err != nil {
				slog.Error("failed to fetch smartrecruiters posting",
					slog.String("id", summary.ID),
					slog.String("err", err.Error()))
				return
			}

			out <- posting.ToModel(companyID)
		}()
	}

	wg.Wait()
}

func (c *Client) fetchPosting(ctx context.Context, slug, id string) (*SRPosting, error) {
	const fn = "SmartRecruiters::fetchPosting"

	url := fmt.Sprintf("%s/%s/postings/%s", c.baseURL, slug, id)
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

	var posting SRPosting
	if err := json.NewDecoder(resp.Body).Decode(&posting); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	return &posting, nil
}
