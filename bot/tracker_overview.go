package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleTrackerOverview(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := b.db.GetOverallTrackerStats(ctx)
	if err != nil {
		log.Printf("tracker overview: %v", err)
		b.followUp(s, i, "An error occurred fetching stats. Please try again later.")
		return
	}

	bar := progressBar(stats.Percentage)
	description := fmt.Sprintf("Licensed Agents: **%d/%d** (%.1f%%)\n\n%s",
		stats.LicensedAgents, stats.TotalAgents, stats.Percentage, bar)

	embed := &discordgo.MessageEmbed{
		Title:       "License Tracker",
		Description: description,
		Color:       0x3498DB,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

// progressBar builds a 10-segment visual progress bar.
func progressBar(percentage float64) string {
	filled := int(percentage / 10)
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", 10-filled)
}
