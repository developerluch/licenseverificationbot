package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleTicketModalSubmit processes the ticket form submission and creates a thread.
func (b *Bot) handleTicketModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	isWAVV := strings.Contains(customID, "wavv")

	// Extract form data
	subject := ""
	category := ""
	description := ""
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			ti := comp.(*discordgo.TextInput)
			switch ti.CustomID {
			case "ticket_subject":
				subject = ti.Value
			case "ticket_category":
				category = ti.Value
			case "ticket_description":
				description = ti.Value
			}
		}
	}

	userID := ""
	userName := "Unknown"
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		userName = i.Member.User.Username
		if i.Member.Nick != "" {
			userName = i.Member.Nick
		}
	}

	// Determine the target channel for the thread
	var parentChannelID string
	var ticketType string
	var embedColor int
	if isWAVV {
		parentChannelID = b.cfg.WAVVTicketChannelID
		ticketType = "WAVV"
		embedColor = 0x00D166
	} else {
		parentChannelID = b.cfg.TicketChannelID
		ticketType = "General"
		embedColor = 0x5865F2
	}

	if parentChannelID == "" {
		respondEphemeral(s, i, "Ticket system is not configured. Please contact an admin.")
		return
	}

	// Create a private thread for the ticket
	threadName := fmt.Sprintf("ticket-%s-%s", strings.ToLower(category), userName)
	if len(threadName) > 100 {
		threadName = threadName[:100]
	}

	thread, err := s.ThreadStartComplex(parentChannelID, &discordgo.ThreadStart{
		Name:                threadName,
		AutoArchiveDuration: 4320, // 3 days
		Type:                discordgo.ChannelTypeGuildPrivateThread,
	})
	if err != nil {
		log.Printf("ticket: failed to create thread: %v", err)
		respondEphemeral(s, i, "Failed to create ticket thread. Please try again or contact an admin.")
		return
	}

	// Post the ticket details and respond to user
	b.postTicketToThread(s, i, thread, userID, userName, subject, category, description, ticketType, embedColor)
}
