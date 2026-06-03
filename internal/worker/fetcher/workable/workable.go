package workable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ErrFetchJobs        = errors.New("failed to fetch jobs")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrDecodeResponse   = errors.New("failed to decode response")
	ErrFetchDetail      = errors.New("failed to fetch detail")
)

type WorkableCompanyConfig struct {
	AccountSlug string `json:"account_slug"`
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
	const fn = "Workable::GetJobs"

	var cfg WorkableCompanyConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	jobs, err := c.listJobs(ctx, cfg.AccountSlug)
	if err != nil {
		return fmt.Errorf("%s:%w", fn, err)
	}

	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			desc, err := c.fetchDescription(ctx, cfg.AccountSlug, job.Shortcode)
			if err != nil {
				slog.Error("failed to fetch workable job description",
					slog.String("shortcode", job.Shortcode),
					slog.String("err", err.Error()))
				return
			}
			job.Description = desc
			out <- job.ToModel(companyID)
		}()
	}

	wg.Wait()
	return nil
}

func (c *Client) listJobs(ctx context.Context, accountSlug string) ([]WorkableJob, error) {
	const fn = "Workable::listJobs"

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}

	url := fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", accountSlug)
	body, _ := json.Marshal(map[string]any{
		"query":      "",
		"department": []string{},
		"location":   []string{},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var result WorkableSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	return result.Results, nil
}

// fetchDescription fetches the LLM-friendly markdown view of a job posting.
// Workable exposes this at /{slug}/jobs/view/{shortcode} and it returns
// the full description as markdown text.
func (c *Client) fetchDescription(ctx context.Context, accountSlug, shortcode string) (string, error) {
	const fn = "Workable::fetchDescription"

	url := fmt.Sprintf("https://apply.workable.com/%s/jobs/view/%s", accountSlug, shortcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s:%w:%w", fn, ErrFetchDetail, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%s:%w:%w", fn, ErrFetchDetail, err)
	}

	return string(body), nil
}
