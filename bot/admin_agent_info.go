package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleAgentCommand routes /agent subcommands (Staff only).
func (b *Bot) handleAgentCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	sub := opts[0]
	switch sub.Name {
	case "info":
		b.handleAgentInfo(s, i, sub.Options)
	case "list":
		b.handleAgentList(s, i, sub.Options)
	case "nudge":
		b.handleAgentNudge(s, i, sub.Options)
	case "promote":
		b.handleAgentPromote(s, i, sub.Options)
	case "stats":
		b.handleAgentStats(s, i)
	case "kick":
		b.handleAgentKick(s, i, sub.Options)
	case "assign-manager":
		b.handleAssignManager(s, i, sub.Options)
	}
}

func (b *Bot) handleAgentInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentInfo: %v", err)
		return
	}
	agent, err := b.db.GetAgent(context.Background(), userIDInt)
	if err != nil || agent == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Agent not found in database.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	member, _ := s.GuildMember(i.GuildID, targetUser.ID)
	embed := buildAgentProfileEmbed(agent, member)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
