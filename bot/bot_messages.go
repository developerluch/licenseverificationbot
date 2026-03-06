package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleMessageCreate auto-deletes user messages in #start-here to keep it clean.
func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Only act on #start-here channel
	if b.cfg.StartHereChannelID == "" || m.ChannelID != b.cfg.StartHereChannelID {
		return
	}
	// Don't delete bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Don't delete other bot messages
	if m.Author.Bot {
		return
	}
	// Delete the user's message
	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		log.Printf("start-here: failed to delete message from %s: %v", m.Author.ID, err)
	}
}
