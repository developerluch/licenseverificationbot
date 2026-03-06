package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
