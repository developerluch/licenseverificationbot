package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleModalSubmit routes modal form submissions.
func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	switch {
	case customID == "vipa:modal_step1":
		b.handleStep1Submit(s, i)
	case customID == "vipa:modal_step2":
		b.handleStep2Submit(s, i)
	case customID == "vipa:modal_step2b":
		b.handleStep2bSubmit(s, i)
	case strings.HasPrefix(customID, "vipa:deny_reason:"):
		b.handleDenyReasonModal(s, i)
	case customID == "vipa:ticket_modal_general", customID == "vipa:ticket_modal_wavv":
		b.handleTicketModalSubmit(s, i)
	}
}
