package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleTicketSetup posts the general support ticket panel in the configured ticket channel.
func (b *Bot) handleTicketSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Staff-only check
	if !b.isStaff(i) {
		respondEphemeral(s, i, "This command is restricted to staff.")
		return
	}

	channelID := b.cfg.TicketChannelID
	if channelID == "" {
		respondEphemeral(s, i, "TICKET_CHANNEL_ID is not configured.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎫 VIPA Support Tickets",
		Description: "Need help? Click the button below to open a support ticket.\n\nOur team will get back to you as soon as possible.\n\n**Please include:**\n• A clear description of your issue\n• Any relevant screenshots\n• Steps you've already tried",
		Color:       0x5865F2, // Discord blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Support System",
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Open Support Ticket",
						Style:    discordgo.PrimaryButton,
						CustomID: "vipa:ticket_open_general",
						Emoji: &discordgo.ComponentEmoji{
							Name: "📩",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("ticket-setup: failed to send panel: %v", err)
		respondEphemeral(s, i, fmt.Sprintf("Failed to post ticket panel: %v", err))
		return
	}

	respondEphemeral(s, i, fmt.Sprintf("✅ General support ticket panel posted in <#%s>.", channelID))
}
