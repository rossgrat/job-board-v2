package greenhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

var (
	ErrUnmarshalConfig  = errors.New("failed to unmarshal config")
	ErrCreateRequest    = errors.New("failed to create request")
	ErrFetchJobs        = errors.New("failed to fetch jobs")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrDecodeResponse   = errors.New("failed to decode response")
)

type GreenhouseCompanyConfig struct {
	BoardSlug string `json:"board_slug"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		baseURL: "https://boards-api.greenhouse.io/v1/boards",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "Greenhouse::GetJobs"

	var cfg GreenhouseCompanyConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	url := fmt.Sprintf("%s/%s/jobs?content=true", g.baseURL, cfg.BoardSlug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var result GreenhouseJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	for _, gj := range result.Jobs {
		out <- gj.ToModel(companyID)
	}

	return nil
}
