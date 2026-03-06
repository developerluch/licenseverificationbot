package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleAgentKick(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	reason := "Removed by staff"
	for _, opt := range opts {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "reason":
			reason = opt.StringValue()
		}
	}

	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentKick: %v", err)
		return
	}

	// DM the user
	b.dmUser(s, targetUser.ID, fmt.Sprintf(
		"You have been removed from the VIPA server.\n\nReason: %s\n\nIf you believe this is an error, contact your upline.", reason))

	// Mark in DB
	b.db.KickAgent(context.Background(), userIDInt, reason)

	// GHL: mark opportunity as lost
	go b.markGHLLost(userIDInt)

	// Kick from server
	err = s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, reason)
	msg := fmt.Sprintf("<@%s> has been removed. Reason: %s", targetUser.ID, reason)
	if err != nil {
		msg += fmt.Sprintf("\n\n⚠️ Discord kick failed: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
