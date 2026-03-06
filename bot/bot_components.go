package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleComponent routes button clicks and other component interactions.
func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	switch {
	// Onboarding buttons
	case customID == "vipa:onboarding_get_started":
		b.handleGetStarted(s, i)
	case customID == "vipa:course_enrolled_yes":
		b.handleCourseEnrolledYes(s, i)
	case customID == "vipa:course_enrolled_no":
		b.handleCourseEnrolledNo(s, i)
	case customID == "vipa:step2_continue":
		b.handleStep2Continue(s, i)
	case customID == "vipa:step2b_continue":
		b.handleStep2bContinue(s, i)

	// Check-in buttons (vipa:checkin:{action}:{week_start})
	case strings.HasPrefix(customID, "vipa:checkin:"):
		b.handleCheckinResponse(s, i)

	// Approval buttons (vipa:approve:{id} or vipa:deny:{id})
	case strings.HasPrefix(customID, "vipa:approve:"), strings.HasPrefix(customID, "vipa:deny:"):
		b.handleApprovalButton(s, i)

	// Setup checklist buttons (vipa:setup:{item_key} or vipa:setup_complete_all)
	case customID == "vipa:setup_complete_all":
		b.handleSetupCompleteAll(s, i)
	case strings.HasPrefix(customID, "vipa:setup:"):
		b.handleSetupItem(s, i)

	// Ticket buttons
	case customID == "vipa:ticket_open_general", customID == "vipa:ticket_open_wavv":
		b.handleTicketOpen(s, i)
	case strings.HasPrefix(customID, "vipa:ticket_close:"):
		b.handleTicketClose(s, i)
	}
}
