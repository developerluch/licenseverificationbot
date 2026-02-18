package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	tls_client_profiles "github.com/bogdanfinn/tls-client/profiles"
)

type CapSolver struct {
	APIKey string
}

func NewCapSolver(apiKey string) *CapSolver {
	if apiKey == "" {
		return nil
	}
	return &CapSolver{APIKey: apiKey}
}

// SolveTurnstile solves a Cloudflare Turnstile challenge via CapSolver API.
func (cs *CapSolver) SolveTurnstile(ctx context.Context, websiteURL, siteKey string) (string, error) {
	// Create a basic TLS client for CapSolver API calls (not the same session as DOI sites)
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(90),
		tls_client.WithClientProfile(tls_client_profiles.Chrome_124),
		tls_client.WithCookieJar(jar),
	}
	client, err := tls_client.NewHttpClient(nil, options...)
	if err != nil {
		return "", fmt.Errorf("capsolver: client error: %w", err)
	}

	// Step 1: Create task
	taskPayload := map[string]interface{}{
		"clientKey": cs.APIKey,
		"task": map[string]interface{}{
			"type":       "AntiTurnstileTaskProxyLess",
			"websiteURL": websiteURL,
			"websiteKey": siteKey,
		},
	}
	taskJSON, _ := json.Marshal(taskPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.capsolver.com/createTask", strings.NewReader(string(taskJSON)))
	if err != nil {
		return "", fmt.Errorf("capsolver: create request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("capsolver: create task failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var createResp struct {
		ErrorID          int    `json:"errorId"`
		ErrorCode        string `json:"errorCode"`
		ErrorDescription string `json:"errorDescription"`
		TaskID           string `json:"taskId"`
	}
	if err := json.Unmarshal(body, &createResp); err != nil {
		return "", fmt.Errorf("capsolver: parse create response: %w", err)
	}
	if createResp.ErrorID != 0 {
		return "", fmt.Errorf("capsolver: create task error: %s - %s", createResp.ErrorCode, createResp.ErrorDescription)
	}

	// Step 2: Poll for result every 3s, max 60s
	pollPayload := map[string]string{
		"clientKey": cs.APIKey,
		"taskId":    createResp.TaskID,
	}
	pollJSON, _ := json.Marshal(pollPayload)

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.capsolver.com/getTaskResult", strings.NewReader(string(pollJSON)))
		if err != nil {
			return "", fmt.Errorf("capsolver: poll request error: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue // Retry on network error
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var pollResp struct {
			ErrorID  int    `json:"errorId"`
			Status   string `json:"status"`
			Solution struct {
				Token string `json:"token"`
			} `json:"solution"`
		}
		if err := json.Unmarshal(body, &pollResp); err != nil {
			continue
		}

		if pollResp.Status == "ready" {
			if pollResp.Solution.Token == "" {
				return "", fmt.Errorf("capsolver: empty token in ready response")
			}
			return pollResp.Solution.Token, nil
		}
		// Status is "processing" -- continue polling
	}

	return "", fmt.Errorf("capsolver: timeout waiting for solution")
}
