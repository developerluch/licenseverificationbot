package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleCheckinResponse processes a check-in button click.
func (b *Bot) handleCheckinResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	// Format: vipa:checkin:{action}:{week_start}
	parts := strings.SplitN(customID, ":", 4)
	if len(parts) < 4 {
		return
	}
	action := parts[2]
	weekStart := parts[3]

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleCheckinResponse: %v", err)
		return
	}

	// Record response
	b.db.RecordCheckinResponse(context.Background(), userIDInt, weekStart, action)

	// Update last_active
	now := time.Now()
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		guildIDInt = b.cfg.GuildIDInt()
	}
	if guildIDInt == 0 {
		guildIDInt = b.cfg.GuildIDInt()
	}
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		LastActive: &now,
	})
	b.db.LogActivity(context.Background(), userIDInt, "checkin_response",
		fmt.Sprintf("Action: %s, Week: %s", action, weekStart))

	var responseMsg string
	switch action {
	case "on_track":
		responseMsg = "\u2705 Great to hear you're on track! Keep up the good work!"
	case "need_help":
		responseMsg = "\u23f8\ufe0f No worries \u2014 we'll connect you with support. Your upline has been notified."
		go b.postCheckinAlert(s, userID, weekStart)
	case "got_licensed":
		responseMsg = "\U0001f393 Amazing! Use `/verify first_name:YourFirst last_name:YourLast state:XX` in the server to verify your license!"
	default:
		responseMsg = "Thanks for checking in!"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    responseMsg,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// postCheckinAlert posts an alert to the hiring log when a student needs help.
func (b *Bot) postCheckinAlert(s *discordgo.Session, userID, weekStart string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postCheckinAlert panic: %v", r)
		}
	}()

	channelID := b.cfg.HiringLogChannelID
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "\u26a0\ufe0f Student Needs Help",
		Description: fmt.Sprintf(
			"<@%s> responded **\"Need Help\"** to their weekly check-in (week of %s).\n\n"+
				"Please follow up with this student.",
			userID, weekStart),
		Color:     0xF39C12,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendWeeklyCheckins sends check-in DMs to students in stages 1-4.
func (b *Bot) sendWeeklyCheckins(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sendWeeklyCheckins panic: %v", r)
		}
	}()

	// Calculate week start (Monday of current week)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)

	students, err := b.db.GetStudentsForCheckin(ctx, weekStart)
	if err != nil {
		log.Printf("Checkins: failed to get students: %v", err)
		return
	}

	if len(students) == 0 {
		log.Println("Checkins: no students need check-in this week")
		return
	}

	weekStartStr := weekStart.Format("2006-01-02")
	log.Printf("Checkins: sending check-ins to %d students for week %s", len(students), weekStartStr)

	for _, student := range students {
		userID := strconv.FormatInt(student.DiscordID, 10)

		// Calculate weeks since join
		weeksIn := int(now.Sub(student.CreatedAt).Hours() / (24 * 7))
		if weeksIn < 1 {
			weeksIn = 1
		}

		name := student.FirstName
		if name == "" {
			name = "Agent"
		}

		embed := buildCheckinEmbed(name, weeksIn)

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "\u2705 On Track",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:checkin:on_track:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "\u23f8\ufe0f Need Help",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:need_help:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "\U0001f393 Got Licensed!",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:got_licensed:%s", weekStartStr),
					},
				},
			},
		}

		channel, err := b.session.UserChannelCreate(userID)
		if err != nil {
			log.Printf("Checkins: can't DM %s: %v", userID, err)
			continue
		}

		_, err = b.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			log.Printf("Checkins: failed to send to %s: %v", userID, err)
			continue
		}

		b.db.RecordCheckinSent(ctx, student.DiscordID, weekStart)
		log.Printf("Checkins: sent week %d check-in to %s (%s)", weeksIn, name, userID)
	}
}

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
