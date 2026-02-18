package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) handleNPNLookup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed: %v", err)
		return
	}

	opts := i.ApplicationCommandData().Options
	optMap := make(map[string]string)
	for _, opt := range opts {
		optMap[opt.Name] = opt.StringValue()
	}

	firstName := strings.TrimSpace(optMap["first_name"])
	lastName := strings.TrimSpace(optMap["last_name"])
	state := strings.ToUpper(strings.TrimSpace(optMap["state"]))

	if firstName == "" || lastName == "" {
		b.followUp(s, i, "Please provide both first and last name.\nExample: `/npn first_name:John last_name:Doe`")
		return
	}

	// If a specific state is given, search just that state
	if state != "" && len(state) == 2 {
		b.npnSingleState(s, i, firstName, lastName, state)
		return
	}

	// Otherwise search across all NAIC states in parallel
	b.npnMultiState(s, i, firstName, lastName)
}

// npnSingleState searches a single state for the agent's NPN.
func (b *Bot) npnSingleState(s *discordgo.Session, i *discordgo.InteractionCreate, firstName, lastName, state string) {
	scraper := b.registry.GetScraper(state)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	results, err := scraper.LookupByName(ctx, firstName, lastName)
	if err != nil {
		b.followUp(s, i, fmt.Sprintf("**Lookup Error** for %s: %v", state, err))
		return
	}

	// Find results with NPN
	var found []scrapers.LicenseResult
	for _, r := range results {
		if r.Found && r.NPN != "" {
			found = append(found, r)
		}
	}

	if len(found) == 0 {
		b.followUp(s, i, fmt.Sprintf("No NPN found for **%s %s** in **%s**.\n\nTry without specifying a state to search all 31 NAIC states.", firstName, lastName, state))
		return
	}

	b.sendNPNResults(s, i, firstName, lastName, found)
}

// npnMultiState searches all 31 NAIC states in parallel for the agent's NPN.
func (b *Bot) npnMultiState(s *discordgo.Session, i *discordgo.InteractionCreate, firstName, lastName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var mu sync.Mutex
	var allResults []scrapers.LicenseResult
	var wg sync.WaitGroup

	// Search all NAIC states in parallel (capped at 10 concurrent)
	sem := make(chan struct{}, 10)

	naicStates := make([]string, 0, len(scrapers.NAICStates))
	for st := range scrapers.NAICStates {
		naicStates = append(naicStates, st)
	}

	// Also add FL, CA, TX since they have NPN in their results
	extraStates := []string{"FL", "CA", "TX"}
	for _, st := range extraStates {
		if !scrapers.NAICStates[st] {
			naicStates = append(naicStates, st)
		}
	}

	for _, st := range naicStates {
		wg.Add(1)
		go func(stateCode string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			scraper := b.registry.GetScraper(stateCode)
			results, err := scraper.LookupByName(ctx, firstName, lastName)
			if err != nil {
				log.Printf("NPN search %s error: %v", stateCode, err)
				return
			}

			mu.Lock()
			for _, r := range results {
				if r.Found && r.NPN != "" {
					allResults = append(allResults, r)
				}
			}
			mu.Unlock()
		}(st)
	}

	wg.Wait()

	if len(allResults) == 0 {
		b.followUp(s, i, fmt.Sprintf(
			"No NPN found for **%s %s** across %d states.\n\n"+
				"This could mean:\n"+
				"- The name doesn't match exactly what's on file\n"+
				"- The license hasn't been processed yet\n"+
				"- The agent is licensed in a state not covered by NAIC\n\n"+
				"Try with an exact name match or specify a state.",
			firstName, lastName, len(naicStates)))
		return
	}

	// Deduplicate by NPN (same agent might appear in multiple states)
	deduped := dedupeByNPN(allResults)
	b.sendNPNResults(s, i, firstName, lastName, deduped)
}

func (b *Bot) sendNPNResults(s *discordgo.Session, i *discordgo.InteractionCreate, firstName, lastName string, results []scrapers.LicenseResult) {
	var fields []*discordgo.MessageEmbedField

	for idx, r := range results {
		if idx >= 5 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  "...",
				Value: fmt.Sprintf("And %d more results. Try narrowing your search with a specific state.", len(results)-5),
			})
			break
		}

		status := "Active"
		if !r.Active {
			status = "Inactive"
		}

		value := fmt.Sprintf(
			"**NPN:** `%s`\n**State:** %s\n**License #:** %s\n**Status:** %s\n**Type:** %s",
			r.NPN, r.State, nvl(r.LicenseNumber, "N/A"), status, nvl(r.LicenseType, "N/A"),
		)
		if r.ExpirationDate != "" {
			value += fmt.Sprintf("\n**Expires:** %s", r.ExpirationDate)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s — %s", nvl(r.FullName, firstName+" "+lastName), r.State),
			Value:  value,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "NPN Lookup Results",
		Description: fmt.Sprintf("Found **%d** license(s) for **%s %s**:", len(results), firstName, lastName),
		Color:       0x3498DB,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "VIPA License Bot • Data from NAIC SBS"},
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("NPN results follow-up failed: %v", err)
	}
}

// dedupeByNPN removes duplicate results with the same NPN, keeping the first (likely home state).
func dedupeByNPN(results []scrapers.LicenseResult) []scrapers.LicenseResult {
	seen := make(map[string]bool)
	var deduped []scrapers.LicenseResult
	for _, r := range results {
		if !seen[r.NPN] {
			seen[r.NPN] = true
			deduped = append(deduped, r)
		}
	}
	return deduped
}
