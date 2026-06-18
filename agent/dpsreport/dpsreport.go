package dpsreport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
// On HTTP 429 it respects the Retry-After header and retries up to 3 times.
func (c *Client) Upload(ctx context.Context, localPath string) (string, error) {
	const maxAttempts = 4
	for attempt := 0; attempt < maxAttempts; attempt++ {
		permalink, retryAfter, err := c.tryUpload(ctx, localPath)
		if err == nil {
			return permalink, nil
		}
		if retryAfter == 0 || attempt == maxAttempts-1 {
			return "", err
		}
		log.Printf("dps.report rate limited; retrying in %s (attempt %d/%d)",
			retryAfter, attempt+1, maxAttempts-1)
		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("unreachable")
}

// tryUpload performs a single upload attempt.
// Returns retryAfter > 0 when the server responded with 429 and the caller should retry.
func (c *Client) tryUpload(ctx context.Context, localPath string) (permalink string, retryAfter time.Duration, err error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", filepath.Base(localPath), err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return "", 0, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", 0, fmt.Errorf("copy file body: %w", err)
	}
	mw.Close()

	url := uploadURL
	if c.userToken != "" {
		url += "&userToken=" + c.userToken
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("POST to dps.report: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", parseRetryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("rate limited by dps.report")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("dps.report returned %s", resp.Status)
	}

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != "" {
		return "", 0, fmt.Errorf("dps.report: %s", result.Error)
	}
	if result.Permalink == "" {
		return "", 0, fmt.Errorf("dps.report returned empty permalink")
	}
	return result.Permalink, 0, nil
}

// parseRetryAfter parses the Retry-After header value (integer seconds).
// Returns 60s if the header is absent or unparseable.
func parseRetryAfter(header string) time.Duration {
	if secs, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 60 * time.Second
}
