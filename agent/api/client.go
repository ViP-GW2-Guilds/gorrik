package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ViP-GW2-Guilds/gorrik/agent/config"
	"github.com/ViP-GW2-Guilds/gorrik/agent/parser"
)

// Client posts parsed log metadata to the Gorrik web API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	dryRun     bool
}

// New creates an API Client from cfg.
func New(cfg *config.Config, dryRun bool) *Client {
	return &Client{
		baseURL: cfg.API.URL,
		apiKey:  cfg.API.Key,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		dryRun: dryRun,
	}
}

// logRequest is the JSON body sent to POST /api/logs.
type logRequest struct {
	Filename      string        `json:"filename"`
	EncounterID   string        `json:"encounter_id"`
	EncounterName string        `json:"encounter_name"`
	Category      string        `json:"category"`
	Subcategory   string        `json:"subcategory"`
	Result        string        `json:"result"`
	Mode          string        `json:"mode"`
	DurationMs    int64         `json:"duration_ms"`
	LoggedAt      time.Time     `json:"logged_at"`
	FileURL       string        `json:"file_url"`
	DpsReportURL  *string       `json:"dps_report_url,omitempty"`
	Players       []playerEntry `json:"players"`
}

type playerEntry struct {
	AccountName   string `json:"account_name"`
	CharacterName string `json:"character_name"`
	Profession    string `json:"profession"`
	EliteSpec     string `json:"elite_spec"`
}

// LogEntry is a row returned by GET /api/logs, used for backfill operations.
type LogEntry struct {
	ID           string  `json:"id"`
	Filename     string  `json:"filename"`
	DpsReportURL *string `json:"dpsReportUrl"`
}

// PostLog sends parsed metadata for a single log to the API.
// dpsReportURL may be empty if no dps.report upload was performed.
// If the API URL is not configured, the call is skipped with a warning.
func (c *Client) PostLog(ctx context.Context, localPath string, meta *parser.LogMetadata, fileURL, dpsReportURL string) error {
	if c.baseURL == "" {
		fmt.Printf("[api] skipping POST for %s — api.url not configured\n", filepath.Base(localPath))
		return nil
	}
	players := make([]playerEntry, len(meta.Players))
	for i, p := range meta.Players {
		players[i] = playerEntry{
			AccountName:   p.AccountName,
			CharacterName: p.CharacterName,
			Profession:    p.Profession,
			EliteSpec:     p.EliteSpec,
		}
	}

	req := logRequest{
		Filename:      filepath.Base(localPath),
		EncounterID:   fmt.Sprintf("%d", meta.EncounterID),
		EncounterName: meta.EncounterName,
		Category:      meta.Category,
		Subcategory:   meta.Subcategory,
		Result:        meta.Result,
		Mode:          meta.Mode,
		DurationMs:    meta.DurationMs,
		LoggedAt:      meta.LoggedAt,
		FileURL:       fileURL,
		Players:       players,
	}
	if dpsReportURL != "" {
		req.DpsReportURL = &dpsReportURL
	}

	if c.dryRun {
		b, _ := json.MarshalIndent(req, "", "  ")
		fmt.Printf("[dry-run] would POST %s/logs:\n%s\n", c.baseURL, b)
		return nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/logs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("POST /logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST /logs: server returned %s", resp.Status)
	}

	return nil
}

// PatchLogDpsURL updates the dps_report_url for a log by its UUID.
func (c *Client) PatchLogDpsURL(ctx context.Context, logID, dpsURL string) error {
	if c.baseURL == "" {
		return fmt.Errorf("api.url not configured")
	}

	body, err := json.Marshal(map[string]string{"dps_report_url": dpsURL})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		c.baseURL+"/logs/"+logID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PATCH /logs/%s: %w", logID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("log %s not found", logID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PATCH /logs/%s: server returned %s", logID, resp.Status)
	}
	return nil
}

// FetchLogsWithoutDpsURL returns all log entries in the DB that have no dps_report_url set.
// It paginates automatically until all results are fetched.
func (c *Client) FetchLogsWithoutDpsURL(ctx context.Context) ([]LogEntry, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("api.url not configured")
	}

	const pageSize = 500
	var all []LogEntry
	offset := 0
	for {
		url := fmt.Sprintf("%s/logs?missing_dps=1&limit=%d&offset=%d", c.baseURL, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GET /logs: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("GET /logs returned %s", resp.Status)
		}

		var result struct {
			Logs  []LogEntry `json:"logs"`
			Total int        `json:"total"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		all = append(all, result.Logs...)
		offset += len(result.Logs)
		if len(result.Logs) < pageSize || offset >= result.Total {
			break
		}
	}
	return all, nil
}
