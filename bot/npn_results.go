package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

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
