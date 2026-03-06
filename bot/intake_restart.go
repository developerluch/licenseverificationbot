package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleRestart reopens the onboarding form for a user (staff only).
func (b *Bot) handleRestart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	// Reset stage to 1 so they can re-do intake
	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleRestart: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleRestart: %v", err)
		return
	}
	stage := db.StageWelcome
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &stage,
	})
	b.db.LogActivity(context.Background(), userIDInt, "restart", "Onboarding restarted by staff")

	// Send welcome DM to the target user
	channel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		embed := buildWelcomeEmbed()
		s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Get Started",
							Style:    discordgo.SuccessButton,
							CustomID: "vipa:onboarding_get_started",
							Emoji:    &discordgo.ComponentEmoji{Name: "🚀"},
						},
					},
				},
			},
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Onboarding restarted for <@%s>. They've been sent a new welcome DM.", targetUser.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
