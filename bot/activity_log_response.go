package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) respondActivityLog(s *discordgo.Session, i *discordgo.InteractionCreate,
	ctx context.Context, userIDInt int64, logged map[string]int, activityTypes []string) {

	// Get weekly totals
	weeklyTotals, err := b.db.GetAgentWeeklyActivity(ctx, userIDInt)
	if err != nil {
		log.Printf("Failed to get weekly activity for %d: %v", userIDInt, err)
	}

	// Build response
	var todayLines, weeklyLines string
	for _, at := range activityTypes {
		if count, ok := logged[at]; ok {
			todayLines += fmt.Sprintf("**%s:** +%d\n", capitalize(at), count)
		}
		if total, ok := weeklyTotals[at]; ok {
			weeklyLines += fmt.Sprintf("**%s:** %d\n", capitalize(at), total)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Activity Logged",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Today", Value: todayLines, Inline: true},
			{Name: "This Week", Value: nvl(weeklyLines, "No activity yet"), Inline: true},
		},
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
