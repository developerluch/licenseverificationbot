package bot

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleLeaderboardCommand handles the /leaderboard command.
func (b *Bot) handleLeaderboardCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		b.followUp(s, i, "Please specify a subcommand: weekly or monthly.")
		return
	}

	sub := opts[0]

	// Extract optional type filter
	activityType := "all"
	for _, opt := range sub.Options {
		if opt.Name == "type" {
			activityType = opt.StringValue()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch sub.Name {
	case "weekly":
		entries, err := b.db.GetWeeklyLeaderboard(ctx, activityType, 10)
		if err != nil {
			log.Printf("leaderboard weekly: %v", err)
			b.followUp(s, i, "An error occurred fetching the leaderboard. Please try again later.")
			return
		}
		embed := b.buildLeaderboardEmbed("Weekly Leaderboard", activityType, entries)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})

	case "monthly":
		entries, err := b.db.GetMonthlyLeaderboard(ctx, activityType, 10)
		if err != nil {
			log.Printf("leaderboard monthly: %v", err)
			b.followUp(s, i, "An error occurred fetching the leaderboard. Please try again later.")
			return
		}
		embed := b.buildLeaderboardEmbed("Monthly Leaderboard", activityType, entries)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}
}
