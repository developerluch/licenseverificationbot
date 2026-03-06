package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"license-bot-go/email"
)

// sendReminders sends DM + email reminders for deadlines approaching.
// Reminders are sent at roughly day 7, 14, and 21 (every 7 days).
func (b *Bot) sendReminders(ctx context.Context, mailer *email.Client) {
	// Get deadlines that haven't had a reminder in the last 6 days
	deadlines, err := b.db.GetPendingDeadlines(ctx, 6*24*time.Hour)
	if err != nil {
		log.Printf("Scheduler: failed to get reminder deadlines: %v", err)
		return
	}

	for _, dl := range deadlines {
		daysLeft := int(math.Ceil(time.Until(dl.DeadlineAt).Hours() / 24))
		if daysLeft > 25 {
			continue // Too early for reminders (first week)
		}

		userID := strconv.FormatInt(dl.DiscordID, 10)

		// Discord DM reminder
		var urgency string
		switch {
		case daysLeft <= 7:
			urgency = "**URGENT:** "
		case daysLeft <= 14:
			urgency = "**Reminder:** "
		default:
			urgency = ""
		}

		msg := fmt.Sprintf(
			"%sYou have **%d days** left to get your insurance license verified.\n\n"+
				"We're checking state records automatically every day. "+
				"As soon as your license shows up, you'll be promoted to **Licensed Producer** automatically.\n\n"+
				"Want to check now? Use `/verify first_name:YourFirst last_name:YourLast state:XX`\n"+
				"Need help? Contact your upline.",
			urgency, daysLeft)

		b.dmUser(b.session, userID, msg)

		// Email reminder (if opted in)
		if mailer != nil {
			agent, err := b.db.GetAgent(ctx, dl.DiscordID)
			if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
				if err := mailer.SendReminder(agent.Email, dl.FirstName+" "+dl.LastName, daysLeft); err != nil {
					log.Printf("Scheduler: email failed for %d: %v", dl.DiscordID, err)
				}
			}
		}

		// Mark reminder sent
		b.db.UpdateReminderSent(ctx, dl.DiscordID)
		log.Printf("Scheduler: sent %d-day reminder to %d (%s %s)", daysLeft, dl.DiscordID, dl.FirstName, dl.LastName)
	}
}
