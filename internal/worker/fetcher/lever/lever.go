package lever

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

type LeverConfig struct {
	Site string `json:"site"`
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
	const fn = "Lever::GetJobs"

	var cfg LeverConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", cfg.Site)
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

	var postings []LeverPosting
	if err := json.NewDecoder(resp.Body).Decode(&postings); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	for _, p := range postings {
		out <- p.ToModel(companyID)
	}

	return nil
}
