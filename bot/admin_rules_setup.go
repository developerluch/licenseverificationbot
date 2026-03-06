package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// handleSetupRules posts the rules embed in #rules (Admin only).
func (b *Bot) handleSetupRules(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	channelID := b.cfg.RulesChannelID
	if channelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "RULES_CHANNEL_ID is not configured.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildRulesEmbed()
	_, err := s.ChannelMessageSendEmbed(channelID, embed)

	msg := "Rules embed posted in <#" + channelID + ">!"
	if err != nil {
		msg = fmt.Sprintf("Failed to post rules: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
