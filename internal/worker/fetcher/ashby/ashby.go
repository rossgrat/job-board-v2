package ashby

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

type AshbyConfig struct {
	Board string `json:"board"`
}

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "Ashby::GetJobs"

	var cfg AshbyConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	url := fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", cfg.Board)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrFetchJobs, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var list AshbyListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	for _, j := range list.Jobs {
		out <- j.ToModel(companyID)
	}

	return nil
}
