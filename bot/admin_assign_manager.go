package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleAssignManager assigns a direct manager to an agent.
func (b *Bot) handleAssignManager(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) < 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please provide both agent and manager.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var agentUser, managerUser *discordgo.User
	for _, opt := range opts {
		switch opt.Name {
		case "agent":
			agentUser = opt.UserValue(s)
		case "manager":
			managerUser = opt.UserValue(s)
		}
	}

	if agentUser == nil || managerUser == nil {
		return
	}

	agentIDInt, err := parseDiscordID(agentUser.ID)
	if err != nil {
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		return
	}
	managerIDInt, err := parseDiscordID(managerUser.ID)
	if err != nil {
		return
	}

	managerName := managerUser.GlobalName
	if managerName == "" {
		managerName = managerUser.Username
	}

	b.db.UpsertAgent(context.Background(), agentIDInt, guildIDInt, db.AgentUpdate{
		DirectManagerDiscordID: &managerIDInt,
		DirectManagerName:      &managerName,
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Assigned <@%s> as direct manager for <@%s>.", managerUser.ID, agentUser.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	log.Printf("Admin: assigned manager %s to agent %s", managerUser.ID, agentUser.ID)
}
