package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleWAVVTicketSetup posts the WAVV support ticket panel in the configured WAVV ticket channel.
func (b *Bot) handleWAVVTicketSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Staff-only check
	if !b.isStaff(i) {
		respondEphemeral(s, i, "This command is restricted to staff.")
		return
	}

	channelID := b.cfg.WAVVTicketChannelID
	if channelID == "" {
		respondEphemeral(s, i, "WAVV_TICKET_CHANNEL_ID is not configured.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📞 WAVV Dialer Support",
		Description: "Having issues with WAVV? Click the button below to open a WAVV support ticket.\n\n**Common WAVV issues we can help with:**\n• Login or account access problems\n• Dialer connection issues\n• Campaign setup help\n• Billing questions\n• Technical troubleshooting",
		Color:       0x00D166, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA WAVV Support",
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Open WAVV Support Ticket",
						Style:    discordgo.SuccessButton,
						CustomID: "vipa:ticket_open_wavv",
						Emoji: &discordgo.ComponentEmoji{
							Name: "📞",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("wavv-ticket-setup: failed to send panel: %v", err)
		respondEphemeral(s, i, fmt.Sprintf("Failed to post WAVV ticket panel: %v", err))
		return
	}

	respondEphemeral(s, i, fmt.Sprintf("✅ WAVV support ticket panel posted in <#%s>.", channelID))
}
