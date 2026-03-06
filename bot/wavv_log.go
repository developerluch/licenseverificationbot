package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleWavvLogCommand processes the /wavv-log slash command.
func (b *Bot) handleWavvLogCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
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
	session := db.WavvSession{
		DiscordID:   userIDInt,
		GuildID:     guildIDInt,
		SessionDate: time.Now(),
	}

	for _, opt := range opts {
		switch opt.Name {
		case "dials":
			session.Dials = int(opt.IntValue())
		case "connections":
			session.Connections = int(opt.IntValue())
		case "talk-mins":
			session.TalkTimeMins = int(opt.IntValue())
		case "appointments":
			session.Appointments = int(opt.IntValue())
		case "callbacks":
			session.Callbacks = int(opt.IntValue())
		case "policies":
			session.Policies = int(opt.IntValue())
		case "notes":
			session.Notes = opt.StringValue()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.db.LogWavvSession(ctx, session); err != nil {
		log.Printf("WAVV log error: %v", err)
		b.followUp(s, i, "Failed to log session. Please try again.")
		return
	}

	connectRate := 0.0
	if session.Dials > 0 {
		connectRate = float64(session.Connections) / float64(session.Dials) * 100
	}

	msg := fmt.Sprintf("**WAVV Session Logged!** 📞\n\n"+
		"📊 **Dials:** %d | **Connections:** %d (%.0f%%)\n"+
		"🗣️ **Talk Time:** %d min | **Appts:** %d\n"+
		"📋 **Callbacks:** %d | **Policies:** %d",
		session.Dials, session.Connections, connectRate,
		session.TalkTimeMins, session.Appointments,
		session.Callbacks, session.Policies)

	if session.Notes != "" {
		msg += fmt.Sprintf("\n📝 **Notes:** %s", session.Notes)
	}

	b.followUp(s, i, msg)
}
