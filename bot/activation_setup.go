package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleSetup shows or manages the agent setup checklist.
func (b *Bot) handleSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetup: %v", err)
		return
	}

	opts := i.ApplicationCommandData().Options
	action := ""
	for _, opt := range opts {
		if opt.Name == "action" {
			action = opt.StringValue()
		}
	}

	if action == "complete" {
		complete, err := b.db.IsSetupComplete(context.Background(), userIDInt)
		if err != nil {
			log.Printf("Setup: check failed: %v", err)
		}
		if !complete {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You haven't completed all setup items yet. Use `/setup start` to see your checklist.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		guildIDInt, err := parseDiscordID(i.GuildID)
		if err != nil {
			log.Printf("handleSetup: %v", err)
			return
		}
		b.activateAgent(s, userIDInt, guildIDInt, i.Member)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "\U0001f389 **Congratulations!** You've completed all setup steps and are now a fully active VIPA agent!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Default: show checklist
	progress, err := b.db.GetSetupProgress(context.Background(), userIDInt)
	if err != nil {
		log.Printf("Setup: get progress failed: %v", err)
		progress = make(map[string]bool)
	}

	// Set stage to setup if not already there
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleSetup: %v", err)
		return
	}
	stage := db.StageSetup
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &stage,
	})

	embed := buildSetupEmbed(i.Member.User.Username, progress)
	rows := buildSetupButtons(progress)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: rows,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
