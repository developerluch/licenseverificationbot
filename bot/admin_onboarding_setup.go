package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// handleOnboardingSetup posts the "Get Started" panel in #start-here (Admin only).
func (b *Bot) handleOnboardingSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	channelID := b.cfg.StartHereChannelID
	if channelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "START_HERE_CHANNEL_ID is not configured.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildWelcomeEmbed()
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
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

	msg := "Get Started panel posted in <#" + channelID + ">!"
	if err != nil {
		msg = fmt.Sprintf("Failed to post panel: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
