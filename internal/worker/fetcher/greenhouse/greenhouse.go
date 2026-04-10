package greenhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/internal/model"
)

var (
	ErrUnmarshalConfig  = errors.New("failed to unmarshal config")
	ErrCreateRequest    = errors.New("failed to create request")
	ErrFetchJobs        = errors.New("failed to fetch jobs")
	ErrFetchOffices     = errors.New("failed to fetch offices")
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

	officeMap, err := g.fetchOfficeMap(ctx, cfg.BoardSlug)
	if err != nil {
		slog.Warn("failed to fetch offices, continuing without resolution", "board", cfg.BoardSlug, "error", err)
	}

	for _, gj := range result.Jobs {
		if officeMap != nil {
			gj.ResolvedOffices = resolveOffices(gj.Offices, officeMap)
		}
		out <- gj.ToModel(companyID)
	}

	return nil
}

// fetchOfficeMap fetches the offices endpoint and returns a flat map of
// office ID to office info. This is used to resolve the child_ids on
// job office entries to actual city names.
func (g *Client) fetchOfficeMap(ctx context.Context, boardSlug string) (map[int]GreenhouseOffice, error) {
	const fn = "Greenhouse::fetchOfficeMap"

	url := fmt.Sprintf("%s/%s/offices", g.baseURL, boardSlug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateRequest, err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrFetchOffices, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnexpectedStatus, fmt.Errorf("%d", resp.StatusCode))
	}

	var result GreenhouseOfficesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrDecodeResponse, err)
	}

	officeMap := make(map[int]GreenhouseOffice)
	var walk func(nodes []GreenhouseOfficeNode)
	walk = func(nodes []GreenhouseOfficeNode) {
		for _, n := range nodes {
			loc := ""
			if n.Location != nil {
				loc = *n.Location
			}
			officeMap[n.ID] = GreenhouseOffice{
				ID:       n.ID,
				Name:     n.Name,
				Location: loc,
			}
			walk(n.ChildOffices)
		}
	}
	walk(result.Offices)

	return officeMap, nil
}

// resolveOffices looks up child_ids for each office on a job and returns
// the child offices that have a non-empty location (i.e. physical offices
// with real city names, not "US-Remote-XX" entries).
func resolveOffices(offices []GreenhouseOffice, officeMap map[int]GreenhouseOffice) []GreenhouseOffice {
	var resolved []GreenhouseOffice
	for _, office := range offices {
		for _, childID := range office.ChildIDs {
			if child, ok := officeMap[childID]; ok && child.Location != "" {
				resolved = append(resolved, child)
			}
		}
	}
	return resolved
}
