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

// handleLogCommand handles the /log command for tracking daily activity.
func (b *Bot) handleLogCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer ephemeral response
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}

	opts := i.ApplicationCommandData().Options
	var notes string
	logged := make(map[string]int)

	activityTypes := []string{"calls", "appointments", "presentations", "policies", "recruits"}

	for _, opt := range opts {
		if opt.Name == "notes" {
			notes = opt.StringValue()
			continue
		}
		for _, at := range activityTypes {
			if opt.Name == at {
				count := int(opt.IntValue())
				if count > 0 {
					logged[at] = count
				}
			}
		}
	}

	if len(logged) == 0 {
		b.followUp(s, i, "Please provide at least one activity count (calls, appointments, presentations, policies, or recruits).")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for actType, count := range logged {
		if err := b.db.LogActivityEntry(ctx, userIDInt, guildIDInt, actType, count, notes); err != nil {
			log.Printf("Activity log: failed to log %s for %d: %v", actType, userIDInt, err)
		}
	}

	// Update last_active
	now := time.Now()
	b.db.UpsertAgent(ctx, userIDInt, guildIDInt, db.AgentUpdate{
		LastActive: &now,
	})

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

	if notes != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Notes: " + notes}
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
