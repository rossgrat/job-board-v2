package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GreenhouseClient struct {
	baseURL    string
	httpClient *http.Client
}

func New() *GreenhouseClient {
	return &GreenhouseClient{
		baseURL: "https://boards-api.greenhouse.io/v1/boards",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GreenhouseClient) GetJobs(ctx context.Context, boardSlug string) ([]GreenhouseJob, error) {
	url := fmt.Sprintf("%s/%s/jobs?content=true", g.baseURL, boardSlug)

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

	return result.Jobs, nil
}
