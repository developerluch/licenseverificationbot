package bot

import (
	"context"
	"log"
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

	b.respondActivityLog(s, i, ctx, userIDInt, logged, activityTypes)
}
