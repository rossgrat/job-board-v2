package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

type GreenhouseCompanyConfig struct {
	BoardSlug string
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

func (g *Client) GetJobs(ctx context.Context, companyID uuid.UUID, config []byte) ([]model.RawJob, error) {
	var cfg GreenhouseCompanyConfig
	err := json.Unmarshal(config, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling greenhouse config: %w", err)
	}

	url := fmt.Sprintf("%s/%s/jobs?content=true", g.baseURL, cfg.BoardSlug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result GreenhouseJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var rawJobs []model.RawJob
	for _, gj := range result.Jobs {
		rawJobs = append(rawJobs, gj.ToModel(companyID))
	}

	return rawJobs, nil
}
