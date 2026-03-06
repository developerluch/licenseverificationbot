package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleTrackerAgency(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agencies, err := b.db.GetAgencyTrackerStats(ctx)
	if err != nil {
		log.Printf("tracker agency: %v", err)
		b.followUp(s, i, "An error occurred fetching agency stats. Please try again later.")
		return
	}

	if len(agencies) == 0 {
		b.followUp(s, i, "No agencies found with assigned agents.")
		return
	}

	var fields []*discordgo.MessageEmbedField
	for _, a := range agencies {
		bar := progressBar(a.Percentage)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   a.Agency,
			Value:  fmt.Sprintf("%d/%d (%.1f%%) %s", a.LicensedAgents, a.TotalAgents, a.Percentage, bar),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     "License Tracker — By Agency",
		Color:     0x3498DB,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}
