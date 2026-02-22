package ghl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://services.leadconnectorhq.com"

// Client is the GoHighLevel API client. Returns nil from NewClient if not configured.
type Client struct {
	apiKey     string
	locationID string
	pipelineID string
	httpClient *http.Client
	stageMap   map[int]string // bot stage -> GHL stage ID
	customFields struct {
		DiscordID string
		Agency    string
		State     string
	}
}

// Config holds the GHL configuration.
type Config struct {
	APIKey     string
	LocationID string
	PipelineID string
	StageMap   map[int]string // bot stage (1-8) -> GHL stage ID

	// Custom field IDs
	CFDiscordID string
	CFAgency    string
	CFState     string
}

// NewClient creates a GHL client. Returns nil if API key is empty (nil-safe pattern).
func NewClient(cfg Config) *Client {
	if cfg.APIKey == "" {
		return nil
	}
	return &Client{
		apiKey:     cfg.APIKey,
		locationID: cfg.LocationID,
		pipelineID: cfg.PipelineID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		stageMap:   cfg.StageMap,
		customFields: struct {
			DiscordID string
			Agency    string
			State     string
		}{
			DiscordID: cfg.CFDiscordID,
			Agency:    cfg.CFAgency,
			State:     cfg.CFState,
		},
	}
}

// do performs an HTTP request to the GHL API.
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ghl: marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("ghl: request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Version", "2021-07-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ghl: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("ghl: read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		bodyStr := string(respBody)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return nil, fmt.Errorf("ghl: HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	return respBody, nil
}
