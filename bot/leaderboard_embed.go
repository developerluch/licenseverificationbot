package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

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
