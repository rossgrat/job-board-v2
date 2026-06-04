package jibe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
)

type JibeConfig struct {
	Host string `json:"host"`
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
	const fn = "Jibe::GetJobs"

	var cfg JibeConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	baseURL := "https://" + strings.TrimSuffix(cfg.Host, "/")

	page := 1
	for {
		list, err := c.fetchPage(ctx, baseURL, page)
		if err != nil {
			return fmt.Errorf("%s:%w", fn, err)
		}
		if len(list.Jobs) == 0 {
			break
		}
		for _, env := range list.Jobs {
			out <- env.Data.ToModel(companyID, baseURL)
		}
		if page*len(list.Jobs) >= list.TotalCount {
			break
		}
		page++
	}

	return nil
}

func (c *Client) fetchPage(ctx context.Context, baseURL string, page int) (*JibeListResponse, error) {
	const fn = "Jibe::fetchPage"

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}

	url := fmt.Sprintf("%s/api/jobs?lang=en-US&page=%d", baseURL, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var list JibeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}
	return &list, nil
}
