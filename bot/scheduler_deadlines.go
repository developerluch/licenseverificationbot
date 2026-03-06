package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
	"license-bot-go/email"
)

// handleExpiredDeadlines notifies admin about users who missed their deadline.
func (b *Bot) handleExpiredDeadlines(ctx context.Context, mailer *email.Client) {
	expired, err := b.db.GetExpiredDeadlines(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get expired deadlines: %v", err)
		return
	}

	for _, dl := range expired {
		userID := strconv.FormatInt(dl.DiscordID, 10)

		// DM the user
		b.dmUser(b.session, userID,
			"**Your 30-day verification deadline has passed.**\n\n"+
				"An admin has been notified. Please contact your upline to discuss next steps.")

		// Email the user (if opted in)
		if mailer != nil {
			agent, err := b.db.GetAgent(ctx, dl.DiscordID)
			if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
				mailer.SendDeadlineExpired(agent.Email, dl.FirstName+" "+dl.LastName)
			}
		}

		// Notify admin channel
		b.notifyAdmin(dl, userID)

		// Mark as admin notified
		b.db.MarkAdminNotified(ctx, dl.DiscordID)
		log.Printf("Scheduler: expired deadline for %d (%s %s), admin notified", dl.DiscordID, dl.FirstName, dl.LastName)
	}
}

func (b *Bot) notifyAdmin(dl db.VerificationDeadline, userID string) {
	channelID := b.cfg.AdminNotificationChannelID
	if channelID == "" {
		channelID = b.cfg.LicenseCheckChannelID
	}
	if channelID == "" {
		channelID = b.cfg.HiringLogChannelID
	}
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "Verification Deadline Expired",
		Description: fmt.Sprintf(
			"<@%s> has not verified their license within 30 days.\n\n"+
				"**Name:** %s %s\n"+
				"**State:** %s\n"+
				"**Status:** %s\n"+
				"**Deadline:** %s\n\n"+
				"Please follow up with this recruit.",
			userID, dl.FirstName, dl.LastName,
			nvl(dl.HomeState, "Unknown"),
			dl.LicenseStatus,
			dl.DeadlineAt.Format("January 2, 2006"),
		),
		Color:     0xE74C3C, // Red
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b.session.ChannelMessageSendEmbed(channelID, embed)
}
