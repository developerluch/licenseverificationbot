package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

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
