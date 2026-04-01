package atomfeed

import (
	"context"
	"encoding/json"
	"encoding/xml"
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
	ErrFetchFeed        = errors.New("failed to fetch feed")
	ErrUnexpectedStatus = errors.New("unexpected status code")
	ErrDecodeFeed       = errors.New("failed to decode feed")
)

type AtomFeedConfig struct {
	FeedURL string `json:"feed_url"`
}

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte, out chan<- model.RawJob) error {
	const fn = "AtomFeed::GetJobs"

	var cfg AtomFeedConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnmarshalConfig, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.FeedURL, nil)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrFetchFeed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var feed AtomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrDecodeFeed, err)
	}

	for _, entry := range feed.Entries {
		out <- entry.ToModel(companyID)
	}

	return nil
}
