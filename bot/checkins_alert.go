package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
