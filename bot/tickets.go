package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// --- Ticket Panel Deployment Commands ---

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

// --- Ticket Creation Handlers ---

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

	// Post the ticket details in the thread
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🎫 %s Support Ticket", ticketType),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Opened By", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
			{Name: "Category", Value: category, Inline: true},
			{Name: "Subject", Value: subject, Inline: false},
			{Name: "Description", Value: description, Inline: false},
		},
		Color:     embedColor,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Ticket created by %s", userName),
		},
	}

	_, err = s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s> — A staff member will be with you shortly.", userID),
		Embeds:  []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Close Ticket",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("vipa:ticket_close:%s", thread.ID),
						Emoji: &discordgo.ComponentEmoji{
							Name: "🔒",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("ticket: failed to post in thread: %v", err)
	}

	// Add the user to the thread
	s.ThreadMemberAdd(thread.ID, userID)

	// Respond to the user
	respondEphemeral(s, i, fmt.Sprintf("✅ Your ticket has been created! Head over to <#%s> to track it.", thread.ID))

	// Log to audit if configured
	if b.cfg.AuditLogChannelID != "" {
		auditEmbed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("New %s Ticket Opened", ticketType),
			Description: fmt.Sprintf("**User:** <@%s>\n**Subject:** %s\n**Category:** %s\n**Thread:** <#%s>",
				userID, subject, category, thread.ID),
			Color:     embedColor,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		s.ChannelMessageSendEmbed(b.cfg.AuditLogChannelID, auditEmbed)
	}
}

// handleTicketClose closes a ticket thread by archiving it.
func (b *Bot) handleTicketClose(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		respondEphemeral(s, i, "Invalid ticket close action.")
		return
	}
	threadID := parts[2]

	closerName := "Unknown"
	if i.Member != nil && i.Member.User != nil {
		closerName = i.Member.User.Username
	}

	// Send closing message
	closeEmbed := &discordgo.MessageEmbed{
		Title:       "🔒 Ticket Closed",
		Description: fmt.Sprintf("This ticket has been closed by **%s**.\n\nThis thread will be archived automatically.", closerName),
		Color:       0x95A5A6,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	s.ChannelMessageSendEmbed(threadID, closeEmbed)

	// Archive the thread
	archived := true
	locked := true
	_, err := s.ChannelEditComplex(threadID, &discordgo.ChannelEdit{
		Archived: &archived,
		Locked:   &locked,
	})
	if err != nil {
		log.Printf("ticket-close: failed to archive thread %s: %v", threadID, err)
		respondEphemeral(s, i, "Failed to close ticket. Please archive the thread manually.")
		return
	}

	respondEphemeral(s, i, "✅ Ticket closed and archived.")
}

// isStaff checks if the interaction user has a staff role.
func (b *Bot) isStaff(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return b.cfg.IsStaff(i.Member.Roles)
}

// respondEphemeral sends an ephemeral message response to an interaction.
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
