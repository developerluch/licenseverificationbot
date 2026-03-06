package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleTicketOpen is called when someone clicks "Open Support Ticket" or "Open WAVV Support Ticket".
func (b *Bot) handleTicketOpen(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	isWAVV := strings.Contains(customID, "wavv")

	var modalTitle, categoryLabel string
	var modalID string
	if isWAVV {
		modalTitle = "WAVV Support Ticket"
		categoryLabel = "Issue Category (login, dialer, billing, other)"
		modalID = "vipa:ticket_modal_wavv"
	} else {
		modalTitle = "Support Ticket"
		categoryLabel = "Issue Category (license, onboarding, technical, other)"
		modalID = "vipa:ticket_modal_general"
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalID,
			Title:    modalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "ticket_subject",
						Label:       "Subject",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "Brief summary of your issue",
						MaxLength:   100,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "ticket_category",
						Label:       categoryLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "e.g. login, billing, technical",
						MaxLength:   50,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "ticket_description",
						Label:       "Describe Your Issue",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Placeholder: "Please describe your issue in detail. Include any error messages, steps to reproduce, and what you've tried so far.",
						MaxLength:   2000,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("ticket-open: modal error: %v", err)
	}
}
