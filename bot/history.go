package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleHistory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer ephemeral
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed on /license-history: %v", err)
		return
	}

	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}

	checks, err := b.db.GetCheckHistory(context.Background(), userIDInt, 5)
	if err != nil {
		log.Printf("History query error: %v", err)
		b.followUp(s, i, "Error fetching history. Try again later.")
		return
	}

	if len(checks) == 0 {
		b.followUp(s, i, "No license checks on file yet. Run `/verify` to get started!")
		return
	}

	var lines []string
	for _, c := range checks {
		emoji := "FAIL"
		if c.Found && c.LicenseStatus != "" && (strings.Contains(strings.ToLower(c.LicenseStatus), "active") || strings.Contains(strings.ToLower(c.LicenseStatus), "valid") || strings.Contains(strings.ToLower(c.LicenseStatus), "verified")) {
			emoji = "PASS"
		}
		status := c.LicenseStatus
		if status == "" {
			status = "Not Found"
		}
		lines = append(lines, fmt.Sprintf("%s **%s** -- %s (%s)", emoji, c.State, status, c.CheckedAt.Format("2006-01-02")))
	}

	// Send as embed
	embed := &discordgo.MessageEmbed{
		Title:       "License Check History",
		Description: strings.Join(lines, "\n"),
		Color:       0x3498DB,
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Follow-up (history) failed: %v", err)
	}
}
