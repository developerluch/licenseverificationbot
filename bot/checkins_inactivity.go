package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

// checkInactivity checks for inactive students and handles warnings/kicks.
func (b *Bot) checkInactivity(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("checkInactivity panic: %v", r)
		}
	}()

	kickWeeks := b.cfg.InactivityKickWeeks
	if kickWeeks <= 0 {
		kickWeeks = 4
	}

	warnWeeks := kickWeeks - 1
	warnThreshold := time.Now().AddDate(0, 0, -warnWeeks*7)
	kickThreshold := time.Now().AddDate(0, 0, -kickWeeks*7)

	// Get all students inactive for at least (kickWeeks-1) weeks
	students, err := b.db.GetInactiveAgents(ctx, warnThreshold)
	if err != nil {
		log.Printf("Inactivity: query failed: %v", err)
		return
	}

	for _, student := range students {
		lastActive := student.CreatedAt
		if student.LastActive != nil {
			lastActive = *student.LastActive
		}

		userID := strconv.FormatInt(student.DiscordID, 10)
		guildID := strconv.FormatInt(student.GuildID, 10)

		if lastActive.Before(kickThreshold) {
			// Kick
			log.Printf("Inactivity: kicking %s (last active: %s)", userID, lastActive.Format("2006-01-02"))

			b.dmUser(b.session, userID,
				"Your membership in VIPA has been removed due to inactivity.\n\n"+
					"If you'd like to rejoin, reach out to your upline or the VIPA team.")

			b.db.KickAgent(ctx, student.DiscordID, "inactivity")

			err := b.session.GuildMemberDeleteWithReason(guildID, userID, "Inactivity \u2014 no check-in response")
			if err != nil {
				log.Printf("Inactivity: kick failed for %s: %v", userID, err)
			}

			if b.cfg.HiringLogChannelID != "" {
				embed := &discordgo.MessageEmbed{
					Title: "\U0001f4e4 Agent Removed \u2014 Inactivity",
					Description: fmt.Sprintf(
						"<@%s> (%s %s) was removed for inactivity.\nLast active: %s",
						userID, student.FirstName, student.LastName,
						lastActive.Format("January 2, 2006")),
					Color:     0xE74C3C,
					Timestamp: time.Now().Format(time.RFC3339),
				}
				b.session.ChannelMessageSendEmbed(b.cfg.HiringLogChannelID, embed)
			}
		} else {
			// Warn
			daysSinceActive := int(time.Since(lastActive).Hours() / 24)
			daysUntilKick := kickWeeks*7 - daysSinceActive
			if daysUntilKick < 1 {
				daysUntilKick = 1
			}

			b.dmUser(b.session, userID, fmt.Sprintf(
				"**Inactivity Warning**\n\n"+
					"You haven't checked in recently. You have **%d days** before your VIPA membership is removed.\n\n"+
					"Reply to your weekly check-in DM or use the server to stay active.",
				daysUntilKick))
		}
	}
}
