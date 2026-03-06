package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"license-bot-go/db"
	"license-bot-go/email"
)

// retryVerifications attempts auto-verify for all pending deadlines.
func (b *Bot) retryVerifications(ctx context.Context, mailer *email.Client) {
	deadlines, err := b.db.GetPendingDeadlines(ctx, 0) // Get all pending
	if err != nil {
		log.Printf("Scheduler: failed to get pending deadlines: %v", err)
		return
	}

	for _, dl := range deadlines {
		if dl.FirstName == "" || dl.LastName == "" || dl.HomeState == "" {
			continue
		}

		verifyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		result := b.performVerification(verifyCtx, dl.FirstName, dl.LastName, dl.HomeState, dl.DiscordID, dl.GuildID)
		cancel()

		if result.Found && result.Match != nil {
			log.Printf("Scheduler: auto-verified %s %s (%d)", dl.FirstName, dl.LastName, dl.DiscordID)
			b.db.MarkDeadlineVerified(ctx, dl.DiscordID)

			// Assign role and notify
			userID := strconv.FormatInt(dl.DiscordID, 10)
			guildID := strconv.FormatInt(dl.GuildID, 10)

			if b.cfg.LicensedAgentRoleID != "" {
				b.session.GuildMemberRoleAdd(guildID, userID, b.cfg.LicensedAgentRoleID)
			}
			if b.cfg.StudentRoleID != "" {
				b.session.GuildMemberRoleRemove(guildID, userID, b.cfg.StudentRoleID)
			}

			// Discord DM
			b.dmUser(b.session, userID, fmt.Sprintf(
				"**Great news!** Your license has been verified for **%s %s** in **%s**!\n\n"+
					"You've been promoted to **Licensed Agent**. Use `/contract` to book your contracting appointment.",
				dl.FirstName, dl.LastName, dl.HomeState))

			// Email notification (if opted in)
			if mailer != nil {
				agent, err := b.db.GetAgent(ctx, dl.DiscordID)
				if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
					licNum := "N/A"
					if result.Match.LicenseNumber != "" {
						licNum = result.Match.LicenseNumber
					}
					if err := mailer.SendVerificationSuccess(agent.Email, dl.FirstName+" "+dl.LastName, dl.HomeState, licNum); err != nil {
						log.Printf("Scheduler: email failed for %d: %v", dl.DiscordID, err)
					}
				}
			}

			// Post to channel
			b.postSchedulerVerifyToChannel(result.Match, dl.HomeState, userID)

			// GHL sync
			go b.syncGHLStage(dl.DiscordID, db.StageVerified)
		}
	}
}
