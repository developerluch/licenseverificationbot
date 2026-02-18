package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"license-bot-go/db"
)

// OnboardingAgent represents the agent data from the Python onboarding bot's API.
type OnboardingAgent struct {
	DiscordID     string `json:"discord_id"`
	FullName      string `json:"full_name"`
	HomeState     string `json:"home_state"`
	PhoneNumber   string `json:"phone_number"`
	LicenseStatus string `json:"license_status"`
	Agency        string `json:"agency"`
	UplineManager string `json:"upline_manager"`
	NPN           string `json:"npn"`
	CurrentStage  int    `json:"current_stage"`
}

// fetchOnboardingAgent calls the Python onboarding bot's API to get agent data.
// This bridges the two bots — when auto-verify fires, we can pull the user's
// name and state from the onboarding bot if our local DB doesn't have it.
func (b *Bot) fetchOnboardingAgent(ctx context.Context, discordID string) (*OnboardingAgent, error) {
	apiURL := b.cfg.OnboardingAPIURL
	apiToken := b.cfg.OnboardingAPIToken

	if apiURL == "" {
		return nil, fmt.Errorf("onboarding API URL not configured")
	}

	// Build request
	url := strings.TrimRight(apiURL, "/") + "/api/v1/agents/" + discordID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("onboarding bridge: build request: %w", err)
	}

	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("onboarding bridge: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("onboarding bridge: read body: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, nil // Agent not found in onboarding system
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("onboarding bridge: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var agent OnboardingAgent
	if err := json.Unmarshal(body, &agent); err != nil {
		return nil, fmt.Errorf("onboarding bridge: parse: %w", err)
	}

	return &agent, nil
}

// syncFromOnboarding pulls agent data from the Python bot and saves it to our DB.
// Returns the first name, last name, and state for verification.
func (b *Bot) syncFromOnboarding(ctx context.Context, discordID string, guildID int64) (firstName, lastName, state string, ok bool) {
	agent, err := b.fetchOnboardingAgent(ctx, discordID)
	if err != nil {
		log.Printf("Onboarding bridge error for %s: %v", discordID, err)
		return "", "", "", false
	}
	if agent == nil {
		return "", "", "", false
	}

	// Parse full name into first + last
	parts := strings.Fields(agent.FullName)
	if len(parts) >= 2 {
		firstName = parts[0]
		lastName = strings.Join(parts[1:], " ")
	} else if len(parts) == 1 {
		firstName = parts[0]
		lastName = parts[0]
	}

	state = strings.ToUpper(strings.TrimSpace(agent.HomeState))

	// Save to our local DB for future lookups
	if firstName != "" {
		discordIDInt, _ := strconv.ParseInt(discordID, 10, 64)
		if discordIDInt != 0 {
			b.db.UpsertAgent(ctx, discordIDInt, guildID, db.AgentUpdate{
				FirstName: &firstName,
				LastName:  &lastName,
				State:     &state,
			})
		}
	}

	log.Printf("Onboarding bridge: synced %s %s (%s) for %s", firstName, lastName, state, discordID)
	return firstName, lastName, state, firstName != "" && state != ""
}
