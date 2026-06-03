package bamboohr

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
	ErrFetchList        = errors.New("failed to fetch list")
	ErrFetchDetail      = errors.New("failed to fetch detail")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrDecodeResponse   = errors.New("failed to decode response")
)

type BambooHRConfig struct {
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

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "BambooHR::GetJobs"

	var cfg BambooHRConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}
	baseURL := fmt.Sprintf("https://%s.bamboohr.com", cfg.Tenant)

	jobs, err := c.listJobs(ctx, baseURL)
	if err != nil {
		return fmt.Errorf("%s:%w", fn, err)
	}

	var wg sync.WaitGroup
	for _, job := range jobs {
		id := job.ID.String()
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := c.limiter.Wait(ctx); err != nil {
				return
			}

			opening, err := c.fetchDetail(ctx, baseURL, id)
			if err != nil {
				slog.Error("failed to fetch bamboohr detail",
					slog.String("tenant", cfg.Tenant),
					slog.String("id", id),
					slog.String("err", err.Error()))
				return
			}

			out <- opening.ToModel(companyID, id)
		}()
	}

	wg.Wait()
	return nil
}

func (c *Client) listJobs(ctx context.Context, baseURL string) ([]BambooHRJob, error) {
	const fn = "BambooHR::listJobs"

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchList, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/careers/list", nil)
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

	var list BambooHRListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	return list.Result, nil
}

func (c *Client) fetchDetail(ctx context.Context, baseURL, id string) (*BambooHRJobOpening, error) {
	const fn = "BambooHR::fetchDetail"

	url := fmt.Sprintf("%s/careers/%s/detail", baseURL, id)
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

	var detail BambooHRDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	return &detail.Result.JobOpening, nil
}
