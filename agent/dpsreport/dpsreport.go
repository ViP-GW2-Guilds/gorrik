package dpsreport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const uploadURL = "https://dps.report/uploadContent?json=1&generator=ei"

// Client uploads EVTC logs to dps.report.
type Client struct {
	userToken  string
	httpClient *http.Client
}

// New creates a Client. userToken may be empty for anonymous uploads.
func New(userToken string) *Client {
	return &Client{
		userToken: userToken,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type uploadResponse struct {
	Permalink string `json:"permalink"`
	Error     string `json:"error"`
}

// Upload sends the log file at localPath to dps.report and returns the permalink.
func (c *Client) Upload(ctx context.Context, localPath string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filepath.Base(localPath), err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", fmt.Errorf("copy file body: %w", err)
	}
	mw.Close()

	url := uploadURL
	if c.userToken != "" {
		url += "&userToken=" + c.userToken
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST to dps.report: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("dps.report returned %s", resp.Status)
	}

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("dps.report: %s", result.Error)
	}
	if result.Permalink == "" {
		return "", fmt.Errorf("dps.report returned empty permalink")
	}
	return result.Permalink, nil
}
