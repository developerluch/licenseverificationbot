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

// handleCheckinResponse processes a check-in button click.
func (b *Bot) handleCheckinResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	// Format: vipa:checkin:{action}:{week_start}
	parts := strings.SplitN(customID, ":", 4)
	if len(parts) < 4 {
		return
	}
	action := parts[2]
	weekStart := parts[3]

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleCheckinResponse: %v", err)
		return
	}

	// Record response
	b.db.RecordCheckinResponse(context.Background(), userIDInt, weekStart, action)

	// Update last_active
	now := time.Now()
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		guildIDInt = b.cfg.GuildIDInt()
	}
	if guildIDInt == 0 {
		guildIDInt = b.cfg.GuildIDInt()
	}
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		LastActive: &now,
	})
	b.db.LogActivity(context.Background(), userIDInt, "checkin_response",
		fmt.Sprintf("Action: %s, Week: %s", action, weekStart))

	var responseMsg string
	switch action {
	case "on_track":
		responseMsg = "\u2705 Great to hear you're on track! Keep up the good work!"
	case "need_help":
		responseMsg = "\u23f8\ufe0f No worries \u2014 we'll connect you with support. Your upline has been notified."
		go b.postCheckinAlert(s, userID, weekStart)
	case "got_licensed":
		responseMsg = "\U0001f393 Amazing! Use `/verify first_name:YourFirst last_name:YourLast state:XX` in the server to verify your license!"
	default:
		responseMsg = "Thanks for checking in!"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    responseMsg,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}
