package atomfeed

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
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

func (c *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte) ([]model.RawJob, error) {
	var cfg AtomFeedConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling atomfeed config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.FeedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feed AtomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decoding atom feed: %w", err)
	}

	var rawJobs []model.RawJob
	for _, entry := range feed.Entries {
		rawJobs = append(rawJobs, entry.ToModel(companyID))
	}

	return rawJobs, nil
}
