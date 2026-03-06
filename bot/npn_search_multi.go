package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

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
