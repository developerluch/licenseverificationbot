package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
