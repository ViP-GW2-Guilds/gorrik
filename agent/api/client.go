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
	Players       []playerEntry `json:"players"`
}

type playerEntry struct {
	AccountName   string `json:"account_name"`
	CharacterName string `json:"character_name"`
	Profession    string `json:"profession"`
	EliteSpec     string `json:"elite_spec"`
}

// PostLog sends parsed metadata for a single log to the API.
// fileURL is the R2 object URL returned by the uploader.
func (c *Client) PostLog(ctx context.Context, localPath string, meta *parser.LogMetadata, fileURL string) error {
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
