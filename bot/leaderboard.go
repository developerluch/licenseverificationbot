package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
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

func (b *Bot) buildLeaderboardEmbed(title, activityType string, entries []db.LeaderboardEntry) *discordgo.MessageEmbed {
	if activityType != "" && activityType != "all" {
		title += " — " + capitalize(activityType)
	}

	if len(entries) == 0 {
		return &discordgo.MessageEmbed{
			Title:       title,
			Description: "No activity logged yet for this period.",
			Color:       0x95A5A6,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	}

	var lines []string
	for _, e := range entries {
		medal := ""
		switch e.Rank {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("**%d.**", e.Rank)
		}
		lines = append(lines, fmt.Sprintf("%s <@%d> — **%d**", medal, e.DiscordID, e.TotalCount))
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: strings.Join(lines, "\n"),
		Color:       0xF1C40F,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
