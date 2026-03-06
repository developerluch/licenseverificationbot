package bot

import (
	"context"
	"fmt"

	"license-bot-go/db"
	"license-bot-go/scrapers"
)

// VerifyResult is the outcome of a license verification attempt.
type VerifyResult struct {
	Found   bool
	Match   *scrapers.LicenseResult
	Error   string
	Message string
}

// performVerification runs a license lookup and returns the result without any Discord interaction.
func (b *Bot) performVerification(ctx context.Context, firstName, lastName, state string, discordID, guildID int64) VerifyResult {
	if state == "" || len(state) != 2 {
		return VerifyResult{Error: "invalid state code"}
	}

	scraper := b.registry.GetScraper(state)

	results, err := scraper.LookupByName(ctx, firstName, lastName)
	if err != nil {
		msg := fmt.Sprintf("Lookup error for %s: %v", state, err)
		return VerifyResult{Error: msg}
	}

	// Find best match: prefer life-licensed active results
	var match *scrapers.LicenseResult
	for idx := range results {
		if results[idx].Found && results[idx].Active && results[idx].IsLifeLicensed() {
			match = &results[idx]
			break
		}
	}
	if match == nil {
		for idx := range results {
			if results[idx].Found && results[idx].Active {
				match = &results[idx]
				break
			}
		}
	}

	if match != nil {
		// Save to DB
		verified := true
		stage := db.StageVerified
		b.db.UpsertAgent(ctx, discordID, guildID, db.AgentUpdate{
			FirstName:       &firstName,
			LastName:        &lastName,
			State:           &state,
			LicenseVerified: &verified,
			LicenseNPN:      &match.NPN,
			CurrentStage:    &stage,
		})
		b.db.SaveLicenseCheck(ctx, db.LicenseCheck{
			DiscordID:      discordID,
			GuildID:        guildID,
			FirstName:      firstName,
			LastName:       lastName,
			State:          state,
			NPN:            match.NPN,
			LicenseNumber:  match.LicenseNumber,
			LicenseType:    match.LicenseType,
			LicenseStatus:  match.Status,
			ExpirationDate: match.ExpirationDate,
			LOAs:           match.LOAs,
			Found:          true,
		})
		return VerifyResult{Found: true, Match: match}
	}

	if len(results) > 0 && results[0].Error != "" {
		return VerifyResult{Error: results[0].Error}
	}

	return VerifyResult{
		Found:   false,
		Message: fmt.Sprintf("No active license found for %s %s in %s", firstName, lastName, state),
	}
}
