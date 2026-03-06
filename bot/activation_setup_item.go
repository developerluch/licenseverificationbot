package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleSetupItem marks a setup item as completed and updates the message.
func (b *Bot) handleSetupItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	itemKey := strings.TrimPrefix(customID, "vipa:setup:")

	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetupItem: %v", err)
		return
	}

	if err := b.db.CompleteSetupItem(context.Background(), userIDInt, itemKey); err != nil {
		log.Printf("Setup: mark item failed: %v", err)
	}
	b.db.LogActivity(context.Background(), userIDInt, "setup_item", fmt.Sprintf("Completed: %s", itemKey))

	// Rebuild the checklist
	progress, err := b.db.GetSetupProgress(context.Background(), userIDInt)
	if err != nil {
		log.Printf("Setup: get progress failed: %v", err)
		progress = make(map[string]bool)
	}

	embed := buildSetupEmbed(i.Member.User.Username, progress)
	rows := buildSetupButtons(progress)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: rows,
		},
	})
}

// handleSetupCompleteAll fires when "Complete Setup" button is clicked.
func (b *Bot) handleSetupCompleteAll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetupCompleteAll: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleSetupCompleteAll: %v", err)
		return
	}

	complete, err := b.db.IsSetupComplete(context.Background(), userIDInt)
	if err != nil || !complete {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Not all setup items are complete yet.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	b.activateAgent(s, userIDInt, guildIDInt, i.Member)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "\U0001f389 **Congratulations!** You've completed all setup steps and are now a fully active VIPA agent!",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}
