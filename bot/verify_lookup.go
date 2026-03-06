package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) performVerifyLookup(s *discordgo.Session, i *discordgo.InteractionCreate,
	firstName, lastName, state, userID string, userIDInt, guildIDInt int64) {

	// Get scraper and lookup
	scraper := b.registry.GetScraper(state)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	results, err := scraper.LookupByName(ctx, firstName, lastName)
	if err != nil {
		log.Printf("Scraper error for %s: %v", state, err)
		msg := fmt.Sprintf("The %s license lookup is temporarily unavailable. Please try again later.", state)
		if url := scraper.ManualLookupURL(); url != "" {
			msg += fmt.Sprintf("\n\nManual lookup: %s", url)
		}
		b.followUp(s, i, msg)
		return
	}

	// Find best match
	var match *scrapers.LicenseResult

	// First pass: prefer life-licensed active results
	for idx := range results {
		if results[idx].Found && results[idx].Active && results[idx].IsLifeLicensed() {
			match = &results[idx]
			break
		}
	}
	// Second pass: any active result
	if match == nil {
		for idx := range results {
			if results[idx].Found && results[idx].Active {
				match = &results[idx]
				break
			}
		}
	}

	// Respond based on result
	b.handleVerifyResult(s, i, match, results, firstName, lastName, state, userID, userIDInt, guildIDInt)
}
