package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) dmVerifyManually(s *discordgo.Session, userID string) {
	b.dmUser(s, userID,
		"Welcome! We see you're a licensed agent. "+
			"Please use `/verify first_name:YourFirst last_name:YourLast state:XX` "+
			"to verify your license and get access to agent channels.")
}

func (b *Bot) handleAutoVerifyFail(s *discordgo.Session, userID, firstName, lastName, state string, userIDInt, guildIDInt int64, result interface{}) {
	log.Printf("Auto-verify: FAILED for %s %s (%s)", firstName, lastName, state)
	b.createVerificationDeadline(userIDInt, guildIDInt, firstName, lastName, state, "pending_licensed")

	b.dmUser(s, userID, fmt.Sprintf(
		"**License Verification Pending**\n\n"+
			"We couldn't automatically verify your license for **%s %s** in **%s**.\n\n"+
			"This could happen if:\n"+
			"- Your name doesn't match what's on file\n"+
			"- Your license is still being processed\n\n"+
			"You have **30 days** to get verified. You can:\n"+
			"1. Use `/verify first_name:YourFirst last_name:YourLast state:XX`\n"+
			"2. Contact your upline for manual verification\n\n"+
			"We'll check again periodically and send you reminders.",
		firstName, lastName, state))
}
